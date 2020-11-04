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
	"testing"

	"arhat.dev/arhat-proto/arhatgopb"
	"arhat.dev/pkg/log"
	"github.com/stretchr/testify/assert"
)

func TestNewServer(t *testing.T) {
	tests := []struct {
		name       string
		listenURLs []EndpointConfig
		expectErr  bool
	}{
		{
			name: "Single Valid",
			listenURLs: []EndpointConfig{
				{
					Listen: "tcp://localhost:0",
				},
			},
		},
		{
			name: "Multiple Valid",
			listenURLs: []EndpointConfig{
				{
					Listen: "tcp://localhost:0",
				},
				{
					Listen: "tcp4://localhost:0",
				},
				{
					Listen: "tcp6://localhost:0",
				},
				{
					Listen: "udp://localhost:0",
				},
				{
					Listen: "udp4://localhost:0",
				},
				{
					Listen: "udp6://localhost:0",
				},
				{
					Listen: "unix:///path/to/sock/file",
				},
			},
		},
		{
			name: "Single Empty",
			listenURLs: []EndpointConfig{
				{
					Listen: "",
				},
			},
			expectErr: true,
		},
		{
			name:       "No ListenURL",
			listenURLs: nil,
			expectErr:  true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			srv, err := NewServer(context.TODO(), log.NoOpLogger, &Config{Endpoints: test.listenURLs})

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
	srv, err := NewServer(context.TODO(), log.NoOpLogger, &Config{Endpoints: []EndpointConfig{{Listen: "tcp://localhost:0"}}})
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
