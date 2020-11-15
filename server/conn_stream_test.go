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
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"arhat.dev/arhat-proto/arhatgopb"
	"arhat.dev/pkg/iohelper"
	"arhat.dev/pkg/log"
	"arhat.dev/pkg/pipenet"
	"github.com/stretchr/testify/assert"

	"arhat.dev/libext/codec"

	// import default codec for test
	_ "arhat.dev/libext/codec/gogoprotobuf"
	_ "arhat.dev/libext/codec/stdjson"

	// import default network support for test
	_ "arhat.dev/pkg/nethelper/piondtls"
	_ "arhat.dev/pkg/nethelper/pipenet"
	_ "arhat.dev/pkg/nethelper/stdnet"
)

func TestStreamConnectionManager_ListenAndServe(t *testing.T) {
	jsonCodec, ok := codec.Get(arhatgopb.CODEC_JSON)
	if !assert.True(t, ok) {
		return
	}

	pbCodec, ok := codec.Get(arhatgopb.CODEC_PROTOBUF)
	if !assert.True(t, ok) {
		return
	}

	tests := []struct {
		name    string
		regName string
		codec   codec.Interface
	}{
		{
			name:    "pb",
			regName: "foo",
			codec:   pbCodec,
		},
		{
			name:    "json",
			regName: "foo",
			codec:   jsonCodec,
		},
	}

	for _, test := range tests {
		t.Run(test.name+"/pipe", func(t *testing.T) {
			serverPath, err := iohelper.TempFilename(os.TempDir(), "*")
			assert.NoError(t, err)

			clientPath := serverPath
			if runtime.GOOS == "windows" {
				serverPath = `\\.\pipe\` + filepath.Base(serverPath)
				clientPath = filepath.Base(serverPath)
			}

			testConnectionManagerListenAndServe(t, &pipenet.PipeAddr{Path: clientPath}, test.regName, test.codec,
				func(handleFunc netConnectionHandleFunc) connectionManager {
					return newStreamConnectionManager(
						context.TODO(), log.NoOpLogger, &pipenet.PipeAddr{Path: serverPath}, nil, handleFunc,
					)
				},
			)
		})

		t.Run(test.name+"/tcp", func(t *testing.T) {
			addr, err := net.ResolveTCPAddr("tcp", "localhost:65530")
			if !assert.NoError(t, err) {
				assert.FailNow(t, "failed to resolve required tcp addr")
				return
			}

			testConnectionManagerListenAndServe(t, addr, test.regName, test.codec,
				func(handleFunc netConnectionHandleFunc) connectionManager {
					return newStreamConnectionManager(context.TODO(), log.NoOpLogger, addr, nil, handleFunc)
				},
			)
		})
	}
}
