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
	"crypto/tls"
	"fmt"
	"time"

	"arhat.dev/arhat-proto/arhatgopb"
	"arhat.dev/pkg/log"

	"arhat.dev/libext"
	"arhat.dev/libext/codec"
	"arhat.dev/libext/extperipheral"

	// import default codec
	_ "arhat.dev/libext/codec/codecjson"
	_ "arhat.dev/libext/codec/codecpb"
)

func ExampleClient() {
	appCtx := context.TODO()
	codec := codec.GetCodec(arhatgopb.CODEC_PROTOBUF)
	client, err := libext.NewClient(
		appCtx,
		arhatgopb.EXTENSION_PERIPHERAL,
		"my-peripheral-extension-name",
		codec,

		nil,
		"unix:///var/run/arhat.sock",
		&tls.Config{},
	)

	if err != nil {
		panic(fmt.Errorf("failed to create client: %w", err))
	}

	ctrl, err := libext.NewController(
		appCtx,
		log.Log.WithName("controller"),
		codec.Marshal,
		extperipheral.NewHandler(log.Log.WithName("handler"), codec.Unmarshal, &examplePeripheralConnector{}),
	)

	for {
		select {
		case <-appCtx.Done():
			return
		default:
			err = client.ProcessNewStream(ctrl.RefreshChannels())
			if err != nil {
				println("error happened when processing data stream", err.Error())
			}
		}
	}
}

type examplePeripheral struct{}

func (d *examplePeripheral) Operate(params map[string]string, data []byte) ([][]byte, error) {
	return [][]byte{[]byte("ok")}, nil
}

func (d *examplePeripheral) CollectMetrics(params map[string]string) ([]*arhatgopb.PeripheralMetricsMsg_Value, error) {
	return []*arhatgopb.PeripheralMetricsMsg_Value{
		{Value: 1.1, Timestamp: time.Now().UnixNano()},
	}, nil
}

func (d *examplePeripheral) Close() {}

type examplePeripheralConnector struct{}

func (c *examplePeripheralConnector) Connect(
	target string, params map[string]string, tlsConfig *arhatgopb.TLSConfig,
) (extperipheral.Peripheral, error) {
	tlsCfg, err := tlsConfig.GetTLSConfig()
	if err != nil {
		return nil, err
	}

	_ = tlsCfg
	return &examplePeripheral{}, nil
}
