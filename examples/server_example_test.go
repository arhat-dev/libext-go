package examples_test

import (
	"context"
	"crypto/tls"
	"sync"

	"arhat.dev/arhat-proto/arhatgopb"
	"arhat.dev/pkg/log"

	"arhat.dev/libext/server"
)

func ExampleServer() {
	extensionSrv, err := server.NewServer(
		context.TODO(),
		log.Log.WithName("name"),
		map[string]*tls.Config{"tcp://localhost:8080": nil},
	)
	if err != nil {
		panic(err)
	}

	s := &exampleServer{
		logger: log.Log.WithName("example-server"),
		srv:    extensionSrv,

		peripheralExtensionHandlers: new(sync.Map),
	}

	// currently we only have peripheral extension available in arhat-proto
	extensionSrv.Handle(arhatgopb.EXTENSION_PERIPHERAL,
		s.CreatePeripheralExtensionHandleFunc,
		"optional", "extension", "names",
	)

	panic(extensionSrv.ListenAndServe())
}

type exampleServer struct {
	logger log.Interface
	srv    *server.Server

	peripheralExtensionHandlers *sync.Map
}

// Start business logic loop
func (s *exampleServer) Start() {
	// TODO: implement your own loop
	for {
		name := "foo"

		logger := s.logger.WithFields(log.String("extension", name))
		// lookup peripheral extension
		v, ok := s.peripheralExtensionHandlers.Load(name)
		if !ok {
			logger.I("peripheral extension not found")
			continue
		}

		ec, ok := v.(*server.ExtensionContext)
		if !ok {
			logger.I("invalid non extension context stored")
		}

		cmd, err := arhatgopb.NewCmd(1, 1, &arhatgopb.PeripheralConnectCmd{
			Target: "data",
			Params: map[string]string{"foo": "bar"},
			Tls:    nil,
		})
		if err != nil {
			logger.I("failed to create peripheral connect cmd", log.Error(err))
			continue
		}

		resp, err := ec.SendCmd(cmd)
		if err != nil {
			logger.I("failed to send peripheral connect cmd", log.Error(err))
			continue
		}

		// TODO: check response and do further process
		_ = resp

		// wait
		<-(chan struct{})(nil)
	}
}

// CreatePeripheralExtensionHandleFunc only creates handle func for peripheral extensions
func (s *exampleServer) CreatePeripheralExtensionHandleFunc(
	extensionName string,
) (server.ExtensionHandleFunc, server.OutOfBandMsgHandleFunc) {
	handleFunc := func(c *server.ExtensionContext) {
		// keep track of all connected peripheral extensions
		_, loaded := s.peripheralExtensionHandlers.LoadOrStore(extensionName, c)
		if loaded {
			// NOTE: extension with same name (in same kind) should never be registered more than once
			//       if you find function returned here, please open an issue to fix it
			return
		}

		defer func() {
			// always close peripheral extension to avoid memory leak
			c.Close()

			// since the extension has been closed, delete the handler of this extension
			s.peripheralExtensionHandlers.Delete(extensionName)
		}()

		for {
			select {
			case <-c.Context.Done():
				return
			}
		}
	}

	oobHandleFunc := func(msg *arhatgopb.Msg) {
		// TODO: route and handle this message according to the msg.Id
		s.logger.I("received out of band message",
			log.String("extension", extensionName),
			log.Uint64("id", msg.Id),
			log.String("msg_type", msg.Kind.String()),
			log.Binary("payload", msg.Payload),
		)
	}

	return handleFunc, oobHandleFunc
}
