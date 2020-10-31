package libext_test

import (
	"context"
	"fmt"
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

func TestClient_ProcessNewStream(t *testing.T) {
	tests := []struct {
		network string
		packet  bool
	}{
		{
			network: "tcp",
			packet:  false,
		},
		{
			network: "tcp4",
			packet:  false,
		},
		{
			network: "tcp6",
			packet:  false,
		},
		{
			network: "udp",
			packet:  true,
		},
		{
			network: "udp4",
			packet:  true,
		},
		{
			network: "udp6",
			packet:  true,
		},
		{
			network: "unix",
			packet:  false,
		},
		{
			network: "unixgram",
			packet:  true,
		},
	}

	for _, test := range tests {
		t.Run(test.network, func(t *testing.T) {
			addr := "localhost:0"
			if strings.HasPrefix(test.network, "unix") {
				if runtime.GOOS == "windows" {
					t.Skipf("ignored unix socket on windows")
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

			var srvAddr string
			if !test.packet {
				l, err := net.Listen(test.network, addr)
				if !assert.NoError(t, err) {
					assert.FailNow(t, "failed to listen stream")
					return
				}

				defer func() {
					_ = l.Close()
				}()

				srvAddr = l.Addr().String()
			} else {
				p, err := net.ListenPacket(test.network, addr)
				if !assert.NoError(t, err) {
					assert.FailNow(t, "failed to listen packet")
					return
				}

				defer func() {
					_ = p.Close()
				}()

				srvAddr = p.LocalAddr().String()
			}

			client, err := libext.NewClient(
				context.TODO(),
				arhatgopb.EXTENSION_PERIPHERAL,
				"test",
				new(codecjson.Codec),
				nil,
				test.network+"://"+srvAddr,
				nil,
			)
			if !assert.NoError(t, err) {
				t.FailNow()
				return
			}

			cmdCh, msgCh := make(chan *arhatgopb.Cmd), make(chan *arhatgopb.Msg)

			go close(msgCh)

			assert.NoError(t, client.ProcessNewStream(cmdCh, msgCh))
			select {
			case <-cmdCh:
			default:
				assert.FailNow(t, "cmdCh not closed")
			}
		})
	}
}
