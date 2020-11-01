package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/url"
	"strings"
	"sync"

	"arhat.dev/arhat-proto/arhatgopb"
	"arhat.dev/pkg/log"
	"golang.org/x/sync/errgroup"

	"arhat.dev/libext/types"
)

func NewServer(
	ctx context.Context,
	logger log.Interface,
	listenURLs map[string]*tls.Config,
) (*Server, error) {
	if len(listenURLs) == 0 {
		return nil, fmt.Errorf("must provide at least one listen url")
	}

	srv := &Server{
		ctx:               ctx,
		logger:            logger,
		extensionManagers: make(map[extensionKey]*ExtensionManager),
		mu:                new(sync.RWMutex),
	}

	for listenURL, tlsConfig := range listenURLs {
		u, err := url.Parse(listenURL)
		if err != nil {
			return nil, fmt.Errorf("invalid listen url: %w", err)
		}

		var connMgr connectionManager
		switch s := strings.ToLower(u.Scheme); s {
		case "tcp", "tcp4", "tcp6": // nolint:goconst
			var addr *net.TCPAddr
			addr, err = net.ResolveTCPAddr(s, u.Host)
			connMgr = newStreamConnectionManager(ctx, logger, addr, tlsConfig, srv.handleNewConn)
		case "udp", "udp4", "udp6": // nolint:goconst
			var addr *net.UDPAddr
			addr, err = net.ResolveUDPAddr(s, u.Host)
			connMgr = newPacketConnectionManager(ctx, logger, addr, tlsConfig, srv.handleNewConn)
		case "unix": // nolint:goconst
			var addr *net.UnixAddr
			addr, err = net.ResolveUnixAddr(s, u.Path)
			connMgr = newStreamConnectionManager(ctx, logger, addr, tlsConfig, srv.handleNewConn)
		//case "fifo":
		//	connector = func() (net.Conn, error) {
		//		return nil, err
		//	}
		default:
			return nil, fmt.Errorf("unsupported protocol %q", u.Scheme)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to resolve %s listen address: %w", u.Scheme, err)
		}

		srv.connectionManagers = append(srv.connectionManagers, connMgr)
	}

	return srv, nil
}

type extensionKey struct {
	kind arhatgopb.ExtensionType
	name string
}

type Server struct {
	ctx    context.Context
	logger log.Interface

	connectionManagers []connectionManager
	extensionManagers  map[extensionKey]*ExtensionManager

	mu *sync.RWMutex
}

// ListenAndServe listen on local addresses and accept connections from extensions
func (s *Server) ListenAndServe() error {
	wg, ctx := errgroup.WithContext(s.ctx)
	_ = ctx
	for _, m := range s.connectionManagers {
		mgr := m
		wg.Go(mgr.ListenAndServe)
	}

	err := wg.Wait()

	for _, m := range s.connectionManagers {
		_ = m.Close()
	}

	return err
}

// Handle extension session with handleFunc, you can specify optional extension names (`extensions`) to restrict
// the handleFunc to these extensions, if no extension name specified, this handleFunc will override all
// existing handleFunc for extensions with this kind
func (s *Server) Handle(kind arhatgopb.ExtensionType, handleFactory ExtensionHandleFuncFactory, extensions ...string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	mgr := NewExtensionManager(s.ctx, s.logger, handleFactory)
	for i, name := range extensions {
		if name == "" {
			return
		}
		s.extensionManagers[extensionKey{
			kind: kind,
			name: extensions[i],
		}] = mgr
	}

	if len(extensions) == 0 {
		// handle all kinds of extensions in this kind
		var toRemove []extensionKey
		for k := range s.extensionManagers {
			if k.kind != kind {
				continue
			}

			toRemove = append(toRemove, extensionKey{
				kind: kind,
				name: k.name,
			})
		}

		for _, k := range toRemove {
			delete(s.extensionManagers, k)
		}

		s.extensionManagers[extensionKey{
			kind: kind,
			name: "",
		}] = mgr
	}
}

func (s *Server) handleNewConn(
	kind arhatgopb.ExtensionType,
	name string,
	codec types.Codec,
	conn io.ReadWriter,
) error {
	s.mu.RLock()
	mgr, ok := s.extensionManagers[extensionKey{
		kind: kind,
		name: name,
	}]
	s.mu.RUnlock()

	if !ok {
		// fallback to general extension
		s.mu.RLock()
		mgr, ok = s.extensionManagers[extensionKey{
			kind: kind,
			name: "",
		}]
		s.mu.RUnlock()
	}

	if !ok {
		return fmt.Errorf("no handler for %s", kind.String())
	}

	return mgr.handleStream(name, codec, conn)
}
