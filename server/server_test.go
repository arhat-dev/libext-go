package server

import (
	"context"
	"crypto/tls"
	"testing"

	"arhat.dev/arhat-proto/arhatgopb"
	"arhat.dev/pkg/log"
	"github.com/stretchr/testify/assert"
)

func TestNewServer(t *testing.T) {
	tests := []struct {
		name       string
		listenURLs map[string]*tls.Config
		expectErr  bool
	}{
		{
			name:       "Single Valid",
			listenURLs: map[string]*tls.Config{"tcp://localhost:0": nil},
		},
		{
			name: "Multiple Valid",
			listenURLs: map[string]*tls.Config{
				"tcp://localhost:0":         nil,
				"tcp4://localhost:0":        nil,
				"tcp6://localhost:0":        nil,
				"udp://localhost:0":         nil,
				"udp4://localhost:0":        nil,
				"udp6://localhost:0":        nil,
				"unix:///path/to/sock/file": nil,
			},
		},
		{
			name:       "Single Empty",
			listenURLs: map[string]*tls.Config{"": nil},
			expectErr:  true,
		},
		{
			name:       "No ListenURL",
			listenURLs: nil,
			expectErr:  true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			srv, err := NewServer(context.TODO(), log.NoOpLogger, test.listenURLs)

			if test.expectErr {
				assert.Error(t, err)
				return
			}

			if !assert.NoError(t, err) {
				assert.FailNow(t, "failed to create required server")
				return
			}

			assert.EqualValues(t, len(test.listenURLs), len(srv.connectionManagers))
			assert.NotNil(t, srv.mu)
			assert.NotNil(t, srv.extensionManagers)
		})
	}
}

func TestServer_Handle(t *testing.T) {
	srv, err := NewServer(context.TODO(), log.NoOpLogger, map[string]*tls.Config{"tcp://localhost:0": nil})
	if !assert.NoError(t, err) {
		assert.FailNow(t, "failed to create required simple server")
		return
	}

	key := extensionKey{
		kind: arhatgopb.EXTENSION_PERIPHERAL,
		name: "foo",
	}
	fallbackKey := extensionKey{
		kind: key.kind,
		name: "",
	}

	// nil and no name
	srv.Handle(key.kind, nil, key.name)
	assert.NotNil(t, srv.extensionManagers[key])
	assert.Nil(t, srv.extensionManagers[key].createHandleFunc)

	// named
	srv.Handle(key.kind, func(extensionName string) (ExtensionHandleFunc, OutOfBandMsgHandleFunc) {
		return nil, nil
	}, key.name, key.name, key.name)
	assert.Len(t, srv.extensionManagers, 1)
	assert.NotNil(t, srv.extensionManagers[key])
	assert.NotNil(t, srv.extensionManagers[key].createHandleFunc)

	// no name
	srv.Handle(key.kind, func(extensionName string) (ExtensionHandleFunc, OutOfBandMsgHandleFunc) {
		return nil, nil
	})
	assert.Len(t, srv.extensionManagers, 1)
	assert.Nil(t, srv.extensionManagers[key])
	assert.NotNil(t, srv.extensionManagers[fallbackKey])
	assert.NotNil(t, srv.extensionManagers[fallbackKey].createHandleFunc)
}
