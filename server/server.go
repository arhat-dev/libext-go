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
	listenURLs []string,
	tlsConfig *tls.Config,
) (*Server, error) {
	if len(listenURLs) == 0 {
		return nil, fmt.Errorf("must provide at least one listen url")
	}

	srv := &Server{
		ctx:               ctx,
		logger:            logger,
		extensionManagers: make(map[arhatgopb.ExtensionType]*extensionManager),
		mu:                new(sync.RWMutex),
	}

	for _, listenURL := range listenURLs {
		u, err := url.Parse(listenURL)
		if err != nil {
			return nil, fmt.Errorf("invalid listen url: %w", err)
		}

		var connMgr connectionManager
		switch s := strings.ToLower(u.Scheme); s {
		case "tcp", "tcp4", "tcp6": // nolint:goconst
			var addr *net.TCPAddr
			addr, err = net.ResolveTCPAddr(s, u.Host)
			connMgr = newStreamConnectionManager(ctx, logger, addr, tlsConfig, srv.handleConnection)
		case "udp", "udp4", "udp6": // nolint:goconst
			var addr *net.UDPAddr
			addr, err = net.ResolveUDPAddr(s, u.Host)
			connMgr = newPacketConnectionManager(ctx, logger, addr, tlsConfig, srv.handleConnection)
		case "unix": // nolint:goconst
			var addr *net.UnixAddr
			addr, err = net.ResolveUnixAddr(s, u.Path)
			connMgr = newStreamConnectionManager(ctx, logger, addr, tlsConfig, srv.handleConnection)
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

type Server struct {
	ctx    context.Context
	logger log.Interface

	connectionManagers []connectionManager
	extensionManagers  map[arhatgopb.ExtensionType]*extensionManager

	mu *sync.RWMutex
}

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

func (s *Server) Handle(kind arhatgopb.ExtensionType, handleFunc ExtensionHandleFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.extensionManagers[kind] = newExtensionManager(s.ctx, s.logger, handleFunc)
}

func (s *Server) handleConnection(
	kind arhatgopb.ExtensionType,
	name string,
	codec types.Codec,
	conn io.ReadWriter,
) error {
	s.mu.RLock()
	mgr, ok := s.extensionManagers[kind]
	s.mu.RUnlock()

	if !ok {
		return fmt.Errorf("no handler for %s", kind.String())
	}

	return mgr.handleStream(name, codec, conn)
}
