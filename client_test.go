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

package libext

import (
	"context"
	"fmt"
	"net"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"arhat.dev/arhat-proto/arhatgopb"
	"arhat.dev/pkg/iohelper"
	"arhat.dev/pkg/nethelper"
	"github.com/stretchr/testify/assert"

	"arhat.dev/libext/codec"
	"arhat.dev/libext/extperipheral"
	"arhat.dev/libext/protoutil"

	// import default codec for test
	_ "arhat.dev/libext/codec/gogoprotobuf"
	_ "arhat.dev/libext/codec/stdjson"

	// import default network support for test
	_ "arhat.dev/pkg/nethelper/piondtls"
	_ "arhat.dev/pkg/nethelper/pipenet"
	_ "arhat.dev/pkg/nethelper/stdnet"
)

type testPacketWriter struct {
	ra   net.Addr
	conn net.PacketConn
}

func (w *testPacketWriter) Write(data []byte) (int, error) {
	return w.conn.WriteTo(data, w.ra)
}

type benchmarkPeripheral struct {
}

func (t *benchmarkPeripheral) Connect(
	ctx context.Context,
	target string,
	params map[string]string,
	tlsConfig *arhatgopb.TLSConfig,
) (extperipheral.Peripheral, error) {
	return nil, nil
}

func TestClient_ProcessNewStream(t *testing.T) {
	type testCase struct {
		network string
		packet  bool
		codec   codec.Interface
	}

	var tests []testCase
	protos := []string{"tcp", "tcp4", "tcp6", "udp", "udp4", "udp6", "unix", "pipe"}
	for _, network := range protos {
		isPkt := strings.HasPrefix(network, "udp")
		for _, c := range []arhatgopb.CodecType{arhatgopb.CODEC_JSON, arhatgopb.CODEC_PROTOBUF} {

			co, ok := codec.Get(c)
			if !assert.True(t, ok, "codec not found "+c.String()) {
				return
			}

			tests = append(tests, testCase{
				network: network,
				packet:  isPkt,
				codec:   co,
			})
		}
	}

	for _, test := range tests {
		t.Run(test.network, func(t *testing.T) {
			var (
				listenAddr string
				err        error
			)

			switch {
			case strings.HasPrefix(test.network, "unix"):
				if runtime.GOOS == "windows" {
					t.Skip("ignored unix socket on windows")
					t.SkipNow()
					return
				}

				listenAddr, err = iohelper.TempFilename(os.TempDir(), fmt.Sprintf("test.%s.*", test.network))
				if !assert.NoError(t, err, "failed to create temporary unix sock file") {
					return
				}
				defer func() {
					_ = os.Remove(listenAddr)
				}()
			case test.network == "pipe":
				switch runtime.GOOS {
				case "windows":
					listenAddr = `\\.\pipe\test-` + test.network
				default:
					listenAddr, err = iohelper.TempFilename(os.TempDir(), "*")
					if !assert.NoError(t, err) {
						return
					}
					defer func() {
						_ = os.Remove(listenAddr)
					}()
				}
			default:
				listenAddr = "localhost:0"
			}

			cmd, err := protoutil.NewCmd(
				test.codec.Marshal, arhatgopb.CMD_PERIPHERAL_CONNECT, 1, 1,
				&arhatgopb.PeripheralOperateCmd{
					Params: map[string]string{"test": "test"},
				},
			)
			if !assert.NoError(t, err) {
				assert.FailNow(t, "failed to create cmd")
				return
			}

			msg, err := protoutil.NewMsg(
				test.codec.Marshal, arhatgopb.MSG_PERIPHERAL_OPERATION_RESULT, 1, 1,
				&arhatgopb.PeripheralOperationResultMsg{
					Result: [][]byte{[]byte("test")},
				},
			)
			if !assert.NoError(t, err) {
				assert.FailNow(t, "failed to create msg")
				return
			}

			var (
				srvAddr     string
				connected   = make(chan struct{})
				clientWrote = make(chan struct{})
			)
			lRaw, err := nethelper.Listen(context.TODO(), nil, test.network, listenAddr, nil)
			if !assert.NoError(t, err, "failed to listen") {
				return
			}

			switch l := lRaw.(type) {
			case net.Listener:
				defer func() {
					_ = l.Close()
				}()

				if test.network == "pipe" && runtime.GOOS == "windows" {
					srvAddr = strings.TrimPrefix(listenAddr, `\\.\pipe\`)
				} else {
					srvAddr = l.Addr().String()
				}
				go func() {
					conn, err2 := l.Accept()
					if !assert.NoError(t, err2) {
						assert.FailNow(t, "failed to accept conn")
						return
					}

					jsonCodec, ok := codec.Get(arhatgopb.CODEC_JSON)
					if !assert.True(t, ok) {
						return
					}

					err2 = jsonCodec.NewDecoder(conn).Decode(&arhatgopb.Msg{})
					assert.NoError(t, err2)

					// encode test command
					enc := test.codec.NewEncoder(conn)
					assert.NoError(t, enc.Encode(cmd))

					close(connected)
					<-clientWrote
					time.Sleep(time.Second)
					_ = conn.Close()
				}()
			case net.PacketConn:
				defer func() {
					_ = l.Close()
				}()

				srvAddr = l.LocalAddr().String()

				go func() {
					buf := make([]byte, 65535)
					_, ra, err2 := l.ReadFrom(buf)
					if !assert.NoError(t, err2) {
						assert.FailNow(t, "failed to read packet")
						return
					}

					conn := &testPacketWriter{
						ra:   ra,
						conn: l,
					}
					enc := test.codec.NewEncoder(conn)
					assert.NoError(t, enc.Encode(cmd))

					close(connected)
					<-clientWrote
				}()
			default:
				assert.Fail(t, "unknown listener type")
			}

			client, err := NewClient(
				context.TODO(),
				arhatgopb.EXTENSION_PERIPHERAL,
				"test",
				test.codec,
				nil,
				test.network+"://"+srvAddr,
				nil,
			)
			if !assert.NoError(t, err) {
				t.FailNow()
				return
			}

			cmdCh, msgCh := make(chan *arhatgopb.Cmd), make(chan *arhatgopb.Msg)

			finished := make(chan struct{})
			go func() {
				assert.NoError(t, client.ProcessNewStream(cmdCh, msgCh))
				close(finished)
			}()

			<-connected

			// receive initial server command
			recvCmd := <-cmdCh
			assert.EqualValues(t, cmd, recvCmd)

			msgCh <- msg

			close(msgCh)

			time.Sleep(time.Second)
			close(clientWrote)

			// wait until client stream handler exited
			<-finished

			select {
			case <-cmdCh:
			default:
				assert.FailNow(t, "cmdCh not closed")
			}
		})
	}
}
