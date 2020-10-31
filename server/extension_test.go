package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"net"
	"testing"
	"time"

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
			msg := &arhatgopb.Msg{
				Kind:    1,
				Id:      1,
				Ack:     100,
				Payload: []byte("msg"),
			}
			msgBytes, err := json.Marshal(msg)
			if !assert.NoError(t, err) {
				assert.FailNow(t, "failed to marshal required msg")
				return
			}

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

			clientConn, srvConn := net.Pipe()
			handleFunc := func(ctx context.Context, cmdCh chan<- *arhatgopb.Cmd, msgCh <-chan *arhatgopb.Msg) error {
				if test.expectErr {
					return io.EOF
				}

				cmdCh <- cmd
				recvMsg := <-msgCh
				assert.EqualValues(t, msg, recvMsg)

				time.Sleep(time.Second)
				_ = clientConn.Close()
				return io.EOF
			}

			mgr := newExtensionManager(context.TODO(), log.NoOpLogger, handleFunc)

			finished := make(chan struct{})
			if !test.expectErr {
				go func() {
					_, err2 := clientConn.Write(msgBytes)
					assert.NoError(t, err2)
				}()

				go func() {
					data, _ := ioutil.ReadAll(clientConn)
					// there will be a new line character after being encoded by json encoder
					assert.EqualValues(t, append(append([]byte{}, cmdBytes...), '\n'), data)
					close(finished)
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

			<-finished
		})
	}
}
