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
	"context"
	"net"
	"testing"

	"arhat.dev/arhat-proto/arhatgopb"
	"arhat.dev/pkg/log"
	"github.com/stretchr/testify/assert"

	"arhat.dev/libext/codec"
	"arhat.dev/libext/types"
)

func TestStreamConnectionManager_ListenAndServe(t *testing.T) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:65530")
	if !assert.NoError(t, err) {
		assert.FailNow(t, "failed to resolve required tcp addr")
		return
	}

	tests := []struct {
		name    string
		regName string
		codec   types.Codec
	}{
		{
			name:    "pb",
			regName: "foo",
			codec:   codec.GetCodec(arhatgopb.CODEC_PROTOBUF),
		},
		{
			name:    "json",
			regName: "foo",
			codec:   codec.GetCodec(arhatgopb.CODEC_JSON),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testConnectionManagerListenAndServe(t, addr, test.regName, test.codec,
				func(handleFunc netConnectionHandleFunc) connectionManager {
					return newStreamConnectionManager(context.TODO(), log.NoOpLogger, addr, nil, handleFunc)
				},
			)
		})
	}
}
