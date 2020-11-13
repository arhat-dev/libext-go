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
	"io"
	"net"
	"testing"
	"time"

	"arhat.dev/arhat-proto/arhatgopb"
	"arhat.dev/pkg/log"
	"github.com/stretchr/testify/assert"

	"arhat.dev/libext"
	"arhat.dev/libext/codec"
	"arhat.dev/libext/types"

	// import default codec for test
	_ "arhat.dev/libext/codec/codecjson"
	_ "arhat.dev/libext/codec/codecpb"
)

func TestPacketConnectionManager_ListenAndServe(t *testing.T) {
	addr, err := net.ResolveUDPAddr("udp", "localhost:65530")
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
					return newPacketConnectionManager(context.TODO(), log.NoOpLogger, addr, nil, handleFunc)
				},
			)
		})
	}
}

func testConnectionManagerListenAndServe(
	t *testing.T,
	addr net.Addr,
	regName string,
	codec types.Codec,
	createMgr func(handleFunc netConnectionHandleFunc) connectionManager,
) {
	time.Sleep(5 * time.Second)

	cmdCh, msgCh := make(chan *arhatgopb.Cmd), make(chan *arhatgopb.Msg)

	handleFunc := func(_ net.Addr, kind arhatgopb.ExtensionType, name string, codec types.Codec, conn io.ReadWriter) error {
		assert.EqualValues(t, regName, name)
		assert.EqualValues(t, codec.Type(), codec.Type())

		close(msgCh)
		return nil
	}

	mgr := createMgr(handleFunc)

	go func() {
		_ = mgr.ListenAndServe()
	}()

	client, err := libext.NewClient(
		context.TODO(),
		arhatgopb.EXTENSION_PERIPHERAL,
		regName,
		codec,
		nil,
		addr.Network()+"://"+addr.String(),
		nil,
	)
	if !assert.NoError(t, err) {
		return
	}

	// TODO: find a better way to connect without error
	time.Sleep(5 * time.Second)
	err = client.ProcessNewStream(cmdCh, msgCh)
	_ = mgr.Close()

	assert.NoError(t, err)
}
