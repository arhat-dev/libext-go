/*
Copyright 2020 The arhat.dev Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package examples_test

import (
	"context"
	"sync"

	"arhat.dev/arhat-proto/arhatgopb"
	"arhat.dev/pkg/log"

	"arhat.dev/libext/server"
	"arhat.dev/libext/util"

	// import default codec
	_ "arhat.dev/libext/codec/codecjson"
	_ "arhat.dev/libext/codec/codecpb"
)

func ExampleServer() {
	extensionSrv, err := server.NewServer(
		context.TODO(),
		log.Log.WithName("name"),
		&server.Config{
			Endpoints: []server.EndpointConfig{
				{
					Listen: "tcp://localhost:8080",
				},
			},
		},
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

		cmd, err := util.NewCmd(
			ec.Codec.Marshal, arhatgopb.CMD_PERIPHERAL_CONNECT, 1, 1,
			&arhatgopb.PeripheralConnectCmd{
				Target: "data",
				Params: map[string]string{"foo": "bar"},
				Tls:    nil,
			},
		)
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
