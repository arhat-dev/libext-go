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
	"io"
	"net"
	"testing"
	"time"

	"arhat.dev/arhat-proto/arhatgopb"
	"arhat.dev/pkg/log"
	"github.com/stretchr/testify/assert"

	"arhat.dev/libext/codec"
)

func TestExtensionManager_handleStream(t *testing.T) {
	tests := []struct {
		name      string
		regName   string
		expectErr bool
	}{
		{
			name:    "Valid First foo",
			regName: "foo",
		},
		// TODO: fix test case for duplicate name
		//{
		//	name:      "Duplicate Second bar",
		//	regName:   "bar",
		//	expectErr: true,
		//},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := codec.GetCodec(arhatgopb.CODEC_JSON)
			cmd := &arhatgopb.Cmd{
				Kind:    100,
				Id:      1,
				Seq:     1,
				Payload: []byte("cmd"),
			}
			cmdBytes, err := c.Marshal(cmd)
			if !assert.NoError(t, err) {
				assert.FailNow(t, "failed to marshal required cmd")
				return
			}

			msgResp := &arhatgopb.Msg{
				Kind:    100,
				Id:      1,
				Ack:     1,
				Payload: []byte("msg"),
			}
			msgRespBytes, err := c.Marshal(msgResp)
			if !assert.NoError(t, err) {
				assert.FailNow(t, "failed to marshal required msg")
				return
			}

			respWrote := make(chan struct{})
			clientExited := make(chan struct{})
			oobMsgWrote := make(chan struct{})
			clientConn, srvConn := net.Pipe()
			handleFunc := func(extensionName string) (ExtensionHandleFunc, OutOfBandMsgHandleFunc) {
				return func(c *ExtensionContext) {
						if test.expectErr {
							return
						}

						go func() {
							_, err2 := clientConn.Write(msgRespBytes)
							assert.NoError(t, err2)

							close(respWrote)
						}()
						_, err2 := c.SendCmd(cmd)
						assert.NoError(t, err2)

						<-respWrote
						<-oobMsgWrote
						<-clientExited
					},
					func(recvMsg *arhatgopb.Msg) {
						assert.EqualValues(t, msgResp, recvMsg)
					}
			}

			mgr := NewExtensionManager(context.TODO(), log.NoOpLogger, handleFunc)

			if !test.expectErr {
				go func() {
					_, err2 := clientConn.Write(msgRespBytes)
					assert.NoError(t, err2)
					close(oobMsgWrote)
				}()

				go func() {
					data := make([]byte, len(cmdBytes)+1)
					_, err2 := io.ReadFull(clientConn, data)
					assert.NoError(t, err2)
					// there will be a new line character after being encoded by json encoder
					assert.EqualValues(t, append(append([]byte{}, cmdBytes...), '\n'), data)

					<-respWrote
					<-oobMsgWrote
					_ = clientConn.Close()
					close(clientExited)
				}()
			} else {
				go func() {
					_ = mgr.HandleStream("bar", c, time.Hour, time.Hour, new(bytes.Buffer))
				}()
			}

			err = mgr.HandleStream(test.regName, c, time.Hour, time.Hour, srvConn)
			_ = srvConn.Close()
			if test.expectErr {
				assert.Error(t, err)
				return
			}

			if !assert.NoError(t, err) {
				assert.FailNow(t, "failed to handle valid stream")
				return
			}

			<-clientExited
		})
	}
}
