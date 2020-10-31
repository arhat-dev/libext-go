package server

import (
	"context"
	"fmt"
	"io"
	"sync"

	"arhat.dev/arhat-proto/arhatgopb"
	"arhat.dev/pkg/log"
	"golang.org/x/sync/errgroup"

	"arhat.dev/libext/types"
)

type ExtensionHandleFunc func(ctx context.Context, cmdCh chan<- *arhatgopb.Cmd, msgCh <-chan *arhatgopb.Msg) error

func newExtension() *extension {
	return &extension{
		cmdCh: make(chan *arhatgopb.Cmd, 1),
		msgCh: make(chan *arhatgopb.Msg, 1),
	}
}

type extension struct {
	cmdCh chan *arhatgopb.Cmd
	msgCh chan *arhatgopb.Msg
}

func (e *extension) close() {
	close(e.msgCh)
}

func newExtensionManager(ctx context.Context, logger log.Interface, handleFunc ExtensionHandleFunc) *extensionManager {
	if logger == nil {
		logger = log.NoOpLogger
	}

	return &extensionManager{
		ctx:    ctx,
		logger: logger,

		registeredExtensions: make(map[string]*extension),

		handle: handleFunc,
		mu:     new(sync.RWMutex),
	}
}

type extensionManager struct {
	ctx    context.Context
	logger log.Interface

	registeredExtensions map[string]*extension

	handle ExtensionHandleFunc

	mu *sync.RWMutex
}

func (m *extensionManager) handleStream(name string, codec types.Codec, stream io.ReadWriter) error {
	// register new extension
	ext := newExtension()
	err := func() error {
		m.mu.Lock()
		defer m.mu.Unlock()

		_, exists := m.registeredExtensions[name]
		if exists {
			return fmt.Errorf("extension %q already registered", name)
		}

		m.registeredExtensions[name] = ext
		return nil
	}()
	if err != nil {
		return fmt.Errorf("failed to register extension: %w", err)
	}

	defer func() {
		m.mu.Lock()
		defer m.mu.Unlock()

		ext.close()
		delete(m.registeredExtensions, name)
	}()

	wg, ctx := errgroup.WithContext(m.ctx)

	wg.Go(func() error {
		err2 := m.handle(ctx, ext.cmdCh, ext.msgCh)
		return err2
	})

	// send commands
	wg.Go(func() error {
		enc := codec.NewEncoder(stream)
		for {
			select {
			case cmd, more := <-ext.cmdCh:
				if !more {
					return io.EOF
				}

				err2 := enc.Encode(cmd)
				if err2 != nil {
					return fmt.Errorf("failed to encode extension command: %w", err2)
				}
			case <-ctx.Done():
				return io.EOF
			}
		}
	})

	// receive messages from extension handler
	wg.Go(func() error {
		dec := codec.NewDecoder(stream)
		for {
			msg := new(arhatgopb.Msg)
			err2 := dec.Decode(msg)
			if err2 != nil {
				if err2 != io.EOF {
					return fmt.Errorf("failed to decode extension message: %w", err2)
				}

				return io.EOF
			}

			select {
			case ext.msgCh <- msg:
			case <-ctx.Done():
				return io.EOF
			}
		}
	})

	err = wg.Wait()
	if err != nil && err != io.EOF {
		return err
	}

	return nil
}
