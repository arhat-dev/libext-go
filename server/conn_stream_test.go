package server

import (
	"context"
	"net"
	"testing"

	"arhat.dev/pkg/log"
	"github.com/stretchr/testify/assert"

	"arhat.dev/libext/codecjson"
	"arhat.dev/libext/codecpb"
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
			testConnectionManagerListenAndServe(t, addr, test.regName, test.codec,
				func(handleFunc netConnectionHandleFunc) connectionManager {
					return newStreamConnectionManager(context.TODO(), log.NoOpLogger, addr, nil, handleFunc)
				},
			)
		})
	}
}
