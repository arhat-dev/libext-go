package server

import (
	"context"
	"testing"

	"arhat.dev/arhat-proto/arhatgopb"
	"arhat.dev/pkg/log"
	"github.com/stretchr/testify/assert"
)

func TestNewServer(t *testing.T) {
	tests := []struct {
		name       string
		listenURLs []string
		expectErr  bool
	}{
		{
			name:       "Single Valid",
			listenURLs: []string{"tcp://localhost:0"},
		},
		{
			name: "Multiple Valid",
			listenURLs: []string{
				"tcp://localhost:0",
				"tcp4://localhost:0",
				"tcp6://localhost:0",
				"udp://localhost:0",
				"udp4://localhost:0",
				"udp6://localhost:0",
				"unix:///path/to/sock/file",
			},
		},
		{
			name:       "Single Empty",
			listenURLs: []string{""},
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
			srv, err := NewServer(context.TODO(), log.NoOpLogger, test.listenURLs, nil)

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
	srv, err := NewServer(context.TODO(), log.NoOpLogger, []string{"tcp://localhost:0"}, nil)
	if !assert.NoError(t, err) {
		assert.FailNow(t, "failed to create required simple server")
		return
	}

	srv.Handle(arhatgopb.EXTENSION_PERIPHERAL, nil)
	assert.NotNil(t, srv.extensionManagers[arhatgopb.EXTENSION_PERIPHERAL])
	assert.Nil(t, srv.extensionManagers[arhatgopb.EXTENSION_PERIPHERAL].handle)

	srv.Handle(arhatgopb.EXTENSION_PERIPHERAL, func(ctx context.Context, cmdCh chan<- *arhatgopb.Cmd, msgCh <-chan *arhatgopb.Msg) error {
		return nil
	})
	assert.NotNil(t, srv.extensionManagers[arhatgopb.EXTENSION_PERIPHERAL])
	assert.NotNil(t, srv.extensionManagers[arhatgopb.EXTENSION_PERIPHERAL].handle)
}
