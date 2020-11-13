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

package extperipheral

import (
	"context"
	"testing"
	"time"

	"arhat.dev/arhat-proto/arhatgopb"
	"arhat.dev/pkg/log"
	"github.com/stretchr/testify/assert"

	"arhat.dev/libext/codec"
	"arhat.dev/libext/types"

	// import default codec
	_ "arhat.dev/libext/codec/codecjson"
	_ "arhat.dev/libext/codec/codecpb"
)

type testPeripheral struct {
}

func (p *testPeripheral) Operate(
	ctx context.Context, params map[string]string, data []byte,
) ([][]byte, error) {
	return [][]byte{[]byte("test")}, nil
}

func (p *testPeripheral) CollectMetrics(
	ctx context.Context, params map[string]string,
) ([]*arhatgopb.PeripheralMetricsMsg_Value, error) {
	return []*arhatgopb.PeripheralMetricsMsg_Value{
		{Value: 1, Timestamp: time.Unix(0, 0).UnixNano()},
	}, nil
}

func (p *testPeripheral) Close(ctx context.Context) {}

type testPeripheralConnector struct {
}

func (p *testPeripheralConnector) Connect(
	ctx context.Context,
	target string,
	params map[string]string,
	tlsConfig *arhatgopb.TLSConfig,
) (Peripheral, error) {
	return &testPeripheral{}, nil
}

func TestHandler_HandleCmd(t *testing.T) {
	tests := []struct {
		name  string
		codec types.Codec
	}{
		{
			name:  "pb",
			codec: codec.GetCodec(arhatgopb.CODEC_PROTOBUF),
		},
		{
			name:  "json",
			codec: codec.GetCodec(arhatgopb.CODEC_JSON),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			h := NewHandler(log.NoOpLogger, test.codec.Unmarshal, &testPeripheralConnector{})
			{
				connectCmdBytes, err := test.codec.Marshal(&arhatgopb.PeripheralConnectCmd{
					Target: "test",
					Params: map[string]string{"test": "test"},
					Tls:    nil,
				})
				assert.NoError(t, err)

				_, msg, err := h.HandleCmd(context.TODO(), 1, 0, arhatgopb.CMD_PERIPHERAL_CONNECT, connectCmdBytes)
				assert.NoError(t, err)
				assert.IsType(t, &arhatgopb.DoneMsg{}, msg)
			}

			{
				operateCmdBytes, err := test.codec.Marshal(&arhatgopb.PeripheralOperateCmd{
					Params: map[string]string{"test": "test"},
					Data:   nil,
				})
				assert.NoError(t, err)

				_, msg, err := h.HandleCmd(context.TODO(), 1, 0, arhatgopb.CMD_PERIPHERAL_OPERATE, operateCmdBytes)
				assert.NoError(t, err)
				assert.IsType(t, &arhatgopb.PeripheralOperationResultMsg{}, msg)
			}

			{
				metricCmdBytes, err := test.codec.Marshal(&arhatgopb.PeripheralMetricsCollectCmd{
					Params: map[string]string{"test": "test"},
				})
				assert.NoError(t, err)

				_, msg, err := h.HandleCmd(context.TODO(), 1, 0, arhatgopb.CMD_PERIPHERAL_COLLECT_METRICS, metricCmdBytes)
				assert.NoError(t, err)
				assert.IsType(t, &arhatgopb.PeripheralMetricsMsg{}, msg)
			}

			{
				closeCmdBytes, err := test.codec.Marshal(&arhatgopb.PeripheralCloseCmd{})
				assert.NoError(t, err)

				_, msg, err := h.HandleCmd(context.TODO(), 1, 0, arhatgopb.CMD_PERIPHERAL_CLOSE, closeCmdBytes)
				assert.NoError(t, err)
				assert.IsType(t, &arhatgopb.DoneMsg{}, msg)
			}

			{
				invalidOperateCmdBytes, err := test.codec.Marshal(&arhatgopb.PeripheralOperateCmd{
					Params: map[string]string{"test": "test"},
				})
				assert.NoError(t, err)

				_, _, err = h.HandleCmd(context.TODO(), 1, 0, arhatgopb.CMD_PERIPHERAL_OPERATE, invalidOperateCmdBytes)
				assert.Error(t, err)
			}
		})
	}
}
