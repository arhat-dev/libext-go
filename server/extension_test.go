package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net"
	"testing"

	"arhat.dev/arhat-proto/arhatgopb"
	"arhat.dev/pkg/log"
	"github.com/stretchr/testify/assert"

	"arhat.dev/libext/codecjson"
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
			cmd := &arhatgopb.Cmd{
				Kind:    100,
				Id:      1,
				Seq:     1,
				Payload: []byte("cmd"),
			}
			cmdBytes, err := json.Marshal(cmd)
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
			msgRespBytes, err := json.Marshal(msgResp)
			if !assert.NoError(t, err) {
				assert.FailNow(t, "failed to marshal required msg")
				return
			}

			msgRespWrote := make(chan struct{})
			clientConn, srvConn := net.Pipe()
			handleFunc := func(extensionName string) (ExtensionHandleFunc, OutOfBandMsgHandleFunc) {
				return func(c *ExtensionContext) {
						if test.expectErr {
							return
						}

						wrote := make(chan struct{})
						go func() {
							_, err2 := clientConn.Write(msgRespBytes)
							assert.NoError(t, err2)

							close(wrote)
						}()
						_, err2 := c.SendCmd(cmd)
						assert.NoError(t, err2)

						<-wrote
						<-msgRespWrote
						_ = clientConn.Close()
					},
					func(recvMsg *arhatgopb.Msg) {
						assert.EqualValues(t, msgResp, recvMsg)
					}
			}

			mgr := NewExtensionManager(context.TODO(), log.NoOpLogger, handleFunc)

			cmdRead := make(chan struct{})
			if !test.expectErr {
				go func() {
					_, err2 := clientConn.Write(msgRespBytes)
					assert.NoError(t, err2)
					close(msgRespWrote)
				}()

				go func() {
					data, _ := ioutil.ReadAll(clientConn)
					// there will be a new line character after being encoded by json encoder
					assert.EqualValues(t, append(append([]byte{}, cmdBytes...), '\n'), data)
					close(cmdRead)
				}()
			} else {
				go func() {
					_ = mgr.handleStream("bar", new(codecjson.Codec), new(bytes.Buffer))
				}()
			}

			err = mgr.handleStream(test.regName, new(codecjson.Codec), srvConn)
			_ = srvConn.Close()
			if test.expectErr {
				assert.Error(t, err)
				return
			}

			if !assert.NoError(t, err) {
				assert.FailNow(t, "failed to handle valid stream")
				return
			}

			<-cmdRead
		})
	}
}
