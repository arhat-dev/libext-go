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
	"arhat.dev/libext/codecjson"
	"arhat.dev/libext/codecpb"
	"arhat.dev/libext/types"
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
			codec:   new(codecpb.Codec),
		},
		{
			name:    "json",
			regName: "foo",
			codec:   new(codecjson.Codec),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			time.Sleep(time.Second)

			testConnectionManagerListenAndServe(t, addr, test.regName, test.codec,
				func(handleFunc netConnectionHandleFunc) connectionManager {
					return newPacketConnectionManager(context.TODO(), log.NoOpLogger, addr, nil, handleFunc)
				},
			)
		})
	}
}

func testConnectionManagerListenAndServe(t *testing.T, addr net.Addr, regName string, codec types.Codec, createMgr func(handleFunc netConnectionHandleFunc) connectionManager) {
	connectionValidated := make(chan struct{})
	cmdCh, msgCh := make(chan *arhatgopb.Cmd), make(chan *arhatgopb.Msg)

	handleFunc := func(kind arhatgopb.ExtensionType, name string, codec types.Codec, conn io.ReadWriter) error {
		assert.EqualValues(t, regName, name)
		assert.EqualValues(t, codec.Type(), codec.Type())

		close(connectionValidated)
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

	go func() {
		<-connectionValidated

		close(msgCh)

		time.Sleep(time.Second)
		_ = mgr.Close()
	}()

	// TODO: find a better way to connect without error
	time.Sleep(5 * time.Second)
	err = client.ProcessNewStream(cmdCh, msgCh)
	assert.NoError(t, err)
}
