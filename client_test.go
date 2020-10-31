package libext_test

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"runtime"
	"strings"
	"testing"

	"arhat.dev/arhat-proto/arhatgopb"
	"github.com/stretchr/testify/assert"

	"arhat.dev/libext"
	"arhat.dev/libext/codecjson"
)

type testPacketWriter struct {
	ra   net.Addr
	conn net.PacketConn
}

func (w *testPacketWriter) Write(data []byte) (int, error) {
	return w.conn.WriteTo(data, w.ra)
}

func TestClient_ProcessNewStream(t *testing.T) {
	tests := []struct {
		network string
		packet  bool
		codec   libext.Codec
	}{
		{
			network: "tcp",
			packet:  false,
			codec:   new(codecjson.Codec),
		},
		{
			network: "tcp4",
			packet:  false,
			codec:   new(codecjson.Codec),
		},
		{
			network: "tcp6",
			packet:  false,
			codec:   new(codecjson.Codec),
		},
		{
			network: "udp",
			packet:  true,
			codec:   new(codecjson.Codec),
		},
		{
			network: "udp4",
			packet:  true,
			codec:   new(codecjson.Codec),
		},
		{
			network: "udp6",
			packet:  true,
			codec:   new(codecjson.Codec),
		},
		{
			network: "unix",
			packet:  false,
			codec:   new(codecjson.Codec),
		},
	}

	for _, test := range tests {
		t.Run(test.network, func(t *testing.T) {
			addr := "localhost:0"
			if strings.HasPrefix(test.network, "unix") {
				if runtime.GOOS == "windows" {
					t.Skip("ignored unix socket on windows")
					t.SkipNow()
					return
				}

				f, err := ioutil.TempFile(os.TempDir(), fmt.Sprintf("test.%s.*", test.network))
				if !assert.NoError(t, err) {
					assert.FailNow(t, "failed to create temporary unix sock file")
					return
				}
				addr = f.Name()

				_ = f.Close()
				_ = os.Remove(addr)

				defer func() {
					_ = os.Remove(addr)
				}()
			}

			cmd, err := arhatgopb.NewCmd(1, 1, &arhatgopb.PeripheralOperateCmd{
				Params: map[string]string{"test": "test"},
			})
			if !assert.NoError(t, err) {
				assert.FailNow(t, "failed to create cmd")
				return
			}

			msg, err := arhatgopb.NewMsg(1, 1, &arhatgopb.PeripheralOperationResultMsg{
				Result: [][]byte{[]byte("test")},
			})
			if !assert.NoError(t, err) {
				assert.FailNow(t, "failed to create msg")
				return
			}

			var (
				srvAddr   string
				netConn   io.Closer
				connected = make(chan struct{})
			)
			if !test.packet {
				l, err2 := net.Listen(test.network, addr)
				if !assert.NoError(t, err2) {
					assert.FailNow(t, "failed to listen stream")
					return
				}

				defer func() {
					_ = l.Close()
				}()

				srvAddr = l.Addr().String()

				go func() {
					conn, err2 := l.Accept()
					if !assert.NoError(t, err2) {
						assert.FailNow(t, "failed to accept conn")
						return
					}
					enc := test.codec.NewEncoder(conn)
					assert.NoError(t, enc.Encode(cmd))

					netConn = conn
					close(connected)
				}()
			} else {
				p, err2 := net.ListenPacket(test.network, addr)
				if !assert.NoError(t, err2) {
					assert.FailNow(t, "failed to listen packet")
					return
				}

				defer func() {
					_ = p.Close()
				}()

				srvAddr = p.LocalAddr().String()

				go func() {
					buf := make([]byte, 65535)
					_, ra, err2 := p.ReadFrom(buf)
					if !assert.NoError(t, err2) {
						assert.FailNow(t, "failed to read packet")
						return
					}

					conn := &testPacketWriter{
						ra:   ra,
						conn: p,
					}
					enc := test.codec.NewEncoder(conn)
					assert.NoError(t, enc.Encode(cmd))

					close(connected)
				}()
			}

			client, err := libext.NewClient(
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

			recvCmd := <-cmdCh
			assert.EqualValues(t, cmd, recvCmd)

			msgCh <- msg

			close(msgCh)

			// wait until client stream handler exited
			<-finished

			if netConn != nil {
				_ = netConn.Close()
			}

			select {
			case <-cmdCh:
			default:
				assert.FailNow(t, "cmdCh not closed")
			}
		})
	}
}
