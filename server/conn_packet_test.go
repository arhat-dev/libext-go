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

	// import default codec for test
	_ "arhat.dev/libext/codec/gogoprotobuf"
	_ "arhat.dev/libext/codec/stdjson"

	// import default network support for test
	_ "arhat.dev/pkg/nethelper/piondtls"
	_ "arhat.dev/pkg/nethelper/pipenet"
	_ "arhat.dev/pkg/nethelper/stdnet"
)

func TestPacketConnectionManager_ListenAndServe(t *testing.T) {
	addr, err := net.ResolveUDPAddr("udp", "localhost:65530")
	if !assert.NoError(t, err) {
		assert.FailNow(t, "failed to resolve required tcp addr")
		return
	}

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
	c codec.Interface,
	createMgr func(handleFunc netConnectionHandleFunc) connectionManager,
) {
	time.Sleep(5 * time.Second)

	cmdCh, msgCh := make(chan *arhatgopb.Cmd), make(chan *arhatgopb.Msg)

	handleFunc := func(_ net.Addr, kind arhatgopb.ExtensionType, name string, c codec.Interface, conn io.ReadWriter) error {
		assert.EqualValues(t, regName, name)
		assert.EqualValues(t, c.Type(), c.Type())

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
		c,
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
