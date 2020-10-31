package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"testing"

	"arhat.dev/arhat-proto/arhatgopb"
	"arhat.dev/pkg/log"
	"github.com/stretchr/testify/assert"
)

func TestBaseConnectionManager_validateConnection(t *testing.T) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if !assert.NoError(t, err) {
		assert.FailNow(t, "failed to resolve required tcp addr")
		return
	}

	tests := []struct {
		name  string
		kind  arhatgopb.ExtensionType
		codec arhatgopb.CodecType
	}{
		{
			name:  "foo",
			kind:  arhatgopb.EXTENSION_PERIPHERAL,
			codec: arhatgopb.CODEC_JSON,
		},
		{
			name:  "bar",
			kind:  arhatgopb.EXTENSION_PERIPHERAL,
			codec: arhatgopb.CODEC_PROTOBUF,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mgr := newBaseConnectionManager(context.TODO(), log.NoOpLogger, addr, nil)

			regMsg, err := arhatgopb.NewMsg(0, 0, &arhatgopb.RegisterMsg{
				Name:          test.name,
				Codec:         test.codec,
				ExtensionType: test.kind,
			})
			if !assert.NoError(t, err) {
				assert.FailNow(t, "failed to create required reg msg")
				return
			}

			// regMsg MUST be encoded in json format for validation
			regMsgBytes, err := json.Marshal(regMsg)
			if !assert.NoError(t, err) {
				return
			}

			kind, name, codec, err := mgr.validateConnection(bytes.NewReader(regMsgBytes))
			if !assert.NoError(t, err) {
				return
			}

			assert.EqualValues(t, test.kind, kind)
			assert.EqualValues(t, test.name, name)
			assert.EqualValues(t, test.codec, codec.Type())
		})
	}
}
