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
package server

import (
	"bytes"
	"context"
	"net"
	"testing"

	"arhat.dev/arhat-proto/arhatgopb"
	"arhat.dev/pkg/log"
	"github.com/stretchr/testify/assert"

	"arhat.dev/libext/codec"
	"arhat.dev/libext/util"
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

			c := codec.GetCodec(arhatgopb.CODEC_JSON)

			regMsg := &arhatgopb.RegisterMsg{
				Name:          test.name,
				Codec:         test.codec,
				ExtensionType: test.kind,
			}
			m, err := util.NewMsg(c.Marshal, util.GetMsgType(regMsg), 0, 0, regMsg)
			if !assert.NoError(t, err) {
				return
			}

			buf := new(bytes.Buffer)
			enc := c.NewEncoder(buf)
			enc.Encode(m)

			kind, name, codec, err := mgr.validateConnection(buf)
			if !assert.NoError(t, err) {
				assert.Errorf(t, err, "failed to validate connection")
				return
			}

			assert.EqualValues(t, test.kind, kind)
			assert.EqualValues(t, test.name, name)
			assert.EqualValues(t, test.codec, codec.Type())
		})
	}
}
