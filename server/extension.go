package server

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"arhat.dev/arhat-proto/arhatgopb"
	"arhat.dev/pkg/log"
	"arhat.dev/pkg/queue"
	"golang.org/x/sync/errgroup"

	"arhat.dev/libext/types"
)

func NewExtensionContext(parent context.Context, name string, sendCmd CmdSendFunc) *ExtensionContext {
	extCtx, cancel := context.WithCancel(parent)
	return &ExtensionContext{
		Context: extCtx,
		Name:    name,
		SendCmd: sendCmd,

		close: cancel,
	}
}

// ExtensionContext of one extension connection
type ExtensionContext struct {
	Context context.Context
	Name    string
	SendCmd CmdSendFunc

	close context.CancelFunc
}

// Close extension connection
func (c *ExtensionContext) Close() {
	c.close()
}

type (
	// Stateful command send func, used to send command to and receive correspond
	// message from the connected extension
	CmdSendFunc func(cmd *arhatgopb.Cmd) (*arhatgopb.Msg, error)

	// Handle message received from extension without previous command request,
	// all messages passed to this function belongs to one extension
	//
	// usually instance of this function type is used to handle extension events
	// 		e.g. lights turned off manually and detected by the extension
	OutOfBandMsgHandleFunc func(msg *arhatgopb.Msg)

	// Handle func for your own business logic
	ExtensionHandleFunc func(c *ExtensionContext)

	// Func factory to create extension specific handle func
	ExtensionHandleFuncFactory func(extensionName string) (ExtensionHandleFunc, OutOfBandMsgHandleFunc)
)

func NewExtensionManager(
	ctx context.Context,
	logger log.Interface,
	handleFuncFactory ExtensionHandleFuncFactory,
) *ExtensionManager {
	if logger == nil {
		logger = log.NoOpLogger
	}

	return &ExtensionManager{
		ctx:              ctx,
		logger:           logger,
		createHandleFunc: handleFuncFactory,

		registeredExtensions: make(map[string]struct{}),

		mu: new(sync.RWMutex),
	}
}

type ExtensionManager struct {
	ctx              context.Context
	logger           log.Interface
	createHandleFunc ExtensionHandleFuncFactory

	registeredExtensions map[string]struct{}

	mu *sync.RWMutex
}

func (m *ExtensionManager) handleStream(name string, codec types.Codec, stream io.ReadWriter) error {
	// register new extension
	err := func() error {
		m.mu.Lock()
		defer m.mu.Unlock()

		_, exists := m.registeredExtensions[name]
		if exists {
			return fmt.Errorf("extension %q already registered", name)
		}

		m.registeredExtensions[name] = struct{}{}
		return nil
	}()
	if err != nil {
		return fmt.Errorf("failed to register extension: %w", err)
	}

	defer func() {
		m.mu.Lock()
		defer m.mu.Unlock()

		delete(m.registeredExtensions, name)
	}()

	wg, ctx := errgroup.WithContext(m.ctx)
	enc := codec.NewEncoder(stream)

	// timeout queue for slow responses
	tq := queue.NewTimeoutQueue()
	tq.Start(ctx.Done())

	msgWait := new(sync.Map)

	closeMsgWait := func(v interface{}) {
		if v == nil {
			return
		}
		mv, ok := v.(*msgWaitValue)
		if !ok {
			return
		}

		mv.close()
	}

	wg.Go(func() error {
		takeCh := tq.TakeCh()
		for {
			select {
			case <-ctx.Done():
				// make everything timeout
				msgWait.Range(func(key, v interface{}) bool {
					closeMsgWait(v)
					msgWait.Delete(key)
					return true
				})
				return io.EOF
			case td := <-takeCh:
				v, _ := msgWait.Load(td.Key)
				closeMsgWait(v)
				msgWait.Delete(td.Key)
			}
		}
	})

	sendCmd := func(cmd *arhatgopb.Cmd) (*arhatgopb.Msg, error) {
		waitV := newMsgWaitValue()
		key := msgWaitKey{
			id:  cmd.Id,
			seq: cmd.Seq,
		}
		_, loaded := msgWait.LoadOrStore(key, waitV)
		if loaded {
			waitV.close()
			return nil, fmt.Errorf("cmd sent before, no response yet")
		}
		_ = tq.OfferWithDelay(key, nil, time.Minute)
		defer func() {
			tq.Remove(key)

			waitV.close()
			msgWait.Delete(key)
		}()
		// send cmd
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			err2 := enc.Encode(cmd)
			if err2 != nil {
				return nil, fmt.Errorf("failed to encode extension command: %w", err2)
			}
		}

		// wait for message response
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case msg, more := <-waitV.msgCh:
			if !more {
				return nil, fmt.Errorf("timeout")
			}

			return msg, nil
		}
	}

	handle, onOutOfBandMsgRecv := m.createHandleFunc(name)

	onMsgRecv := func(msg *arhatgopb.Msg) {
		key := msgWaitKey{
			id:  msg.Id,
			seq: msg.Ack,
		}

		v, ok := msgWait.Load(key)
		if !ok || v == nil {
			onOutOfBandMsgRecv(msg)
			return
		}

		mv, ok := v.(*msgWaitValue)
		if !ok {
			onOutOfBandMsgRecv(msg)
			return
		}

		mv.send(ctx.Done(), msg)
	}

	wg.Go(func() error {
		handle(NewExtensionContext(ctx, name, sendCmd))

		// return trivial error to trigger context exit
		return io.EOF
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
			case <-ctx.Done():
				return io.EOF
			default:
				onMsgRecv(msg)
			}
		}
	})

	err = wg.Wait()
	if err != nil && err != io.EOF {
		return err
	}

	return nil
}
