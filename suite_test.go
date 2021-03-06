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
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"arhat.dev/arhat-proto/arhatgopb"
	"arhat.dev/pkg/iohelper"
	"arhat.dev/pkg/log"
	"github.com/stretchr/testify/assert"

	"arhat.dev/libext/codec"
	"arhat.dev/libext/extperipheral"
	"arhat.dev/libext/protoutil"
	"arhat.dev/libext/server"

	// import default codec for test
	_ "arhat.dev/libext/codec/gogoprotobuf"
	_ "arhat.dev/libext/codec/stdjson"

	// import default network support for test
	_ "arhat.dev/pkg/nethelper/piondtls"
	_ "arhat.dev/pkg/nethelper/pipenet"
	_ "arhat.dev/pkg/nethelper/stdnet"
)

func BenchmarkSuite(b *testing.B) {
	type benchCase struct {
		network string
		host    string
		codec   codec.Interface
		tls     bool
	}

	jsonCodec, ok := codec.Get(arhatgopb.CODEC_JSON)
	if !assert.True(b, ok) {
		return
	}

	pbCodec, ok := codec.Get(arhatgopb.CODEC_PROTOBUF)
	if !assert.True(b, ok) {
		return
	}

	var (
		protos = []string{"tcp4", "tcp6", "udp4", "udp6", "pipe"}
		codecs = []codec.Interface{
			jsonCodec,
			pbCodec,
		}
	)

	if runtime.GOOS != "windows" {
		protos = append(protos, "unix")
	}

	var benchmarks []benchCase
	for i := range protos {
		for j := range codecs {
			host := "127.0.0.1"
			if strings.Contains(protos[i], "6") {
				host = "::1"
			}

			benchmarks = append(benchmarks,
				benchCase{
					network: protos[i],
					host:    host,
					codec:   codecs[j],
					tls:     false,
				},
				benchCase{
					network: protos[i],
					host:    host,
					codec:   codecs[j],
					tls:     true,
				},
			)
		}
	}

	for i, bm := range benchmarks {
		port := 50000 + i*100
		suffix := ""
		if bm.tls {
			suffix = "-tls"
		}

		b.Run(bm.network+suffix+"/"+bm.codec.Type().String(), func(b *testing.B) {
			b.StopTimer()

			port++
			var (
				addr string
				port = strconv.FormatInt(int64(port), 10)
			)
			switch {
			case bm.network == "unix",
				bm.network == "pipe" && runtime.GOOS != "windows":
				filename, err := iohelper.TempFilename(os.TempDir(), "bench-unix-*.sock")
				if err != nil {
					b.Errorf("failed to create temporary unix socket file: %v", err)
					return
				}
				addr = bm.network + "://" + filename
			case bm.network == "pipe" && runtime.GOOS == "windows":
				addr = bm.network + "://benchmark-" + port
			default:
				addr = bm.network + "://" + net.JoinHostPort(bm.host, port)
			}

			var (
				serverTLS *tls.Config
				clientTLS *tls.Config
			)
			if bm.tls {
				caBytes, err := ioutil.ReadFile("testdata/ca-cert.pem")
				if err != nil {
					b.Errorf("failed to load ca cert: %v", err)
					return
				}

				serverCert, err := tls.LoadX509KeyPair("testdata/tls-cert.pem", "testdata/tls-key.pem")
				if err != nil {
					b.Errorf("failed to load server tls cert pair: %v", err)
					return
				}

				clientCert, err := tls.LoadX509KeyPair("testdata/client-tls-cert.pem", "testdata/client-tls-key.pem")
				if err != nil {
					b.Errorf("failed to load client tls cert pair: %v", err)
					return
				}

				cp := x509.NewCertPool()
				cp.AppendCertsFromPEM(caBytes)
				serverTLS = &tls.Config{
					ClientCAs:          cp,
					Certificates:       []tls.Certificate{serverCert},
					ClientAuth:         tls.RequireAndVerifyClientCert,
					InsecureSkipVerify: false,
				}

				clientTLS = &tls.Config{
					RootCAs:            cp,
					Certificates:       []tls.Certificate{clientCert},
					ServerName:         "localhost",
					InsecureSkipVerify: false,
				}
			}

			srv, err := server.NewServer(context.TODO(), log.NoOpLogger, &server.Config{
				Endpoints: []server.EndpointConfig{
					{
						Listen:            addr,
						KeepaliveInterval: time.Hour,
						MessageTimeout:    time.Hour,
						TLS:               serverTLS,
					},
				},
			})

			if err != nil {
				b.Errorf("failed to create server: %v", err)
				return
			}

			client, err := NewClient(
				context.TODO(),
				arhatgopb.EXTENSION_PERIPHERAL,
				"benchmark", bm.codec,
				nil, addr, clientTLS,
			)
			if err != nil {
				b.Errorf("failed to create client: %v", err)
				return
			}

			ctrl, err := NewController(
				context.TODO(),
				log.NoOpLogger,
				bm.codec.Marshal,
				extperipheral.NewHandler(log.NoOpLogger, bm.codec.Unmarshal, new(benchmarkPeripheral)),
			)
			if err != nil {
				b.Errorf("failed to create controller: %v", err)
				return
			}

			err = ctrl.Start()
			if err != nil {
				b.Errorf("failed to start controller: %v", err)
				return
			}

			srv.Handle(arhatgopb.EXTENSION_PERIPHERAL, func(extensionName string) (server.ExtensionHandleFunc, server.OutOfBandMsgHandleFunc) {
				return func(c *server.ExtensionContext) {
						defer func() {
							c.Close()
						}()

						b.ReportAllocs()
						b.ResetTimer()
						b.StartTimer()

						for idx := 0; idx < b.N; idx++ {
							cmd, err2 := protoutil.NewCmd(
								bm.codec.Marshal, arhatgopb.CMD_PERIPHERAL_CONNECT, uint64(idx), 1,
								&arhatgopb.PeripheralConnectCmd{
									Target: "benchmark",
									Params: map[string]string{"test": "benchmark"},
									Tls:    nil,
								},
							)
							if err2 != nil {
								b.Errorf("unexpected cmd creation error: %v", err2)
								return
							}

							msg, err2 := c.SendCmd(cmd, true)
							if err2 != nil || msg.Kind != arhatgopb.MSG_DONE {
								if msg != nil {
									println(string(msg.Payload))
								}
								b.Errorf("unexpected message: %v", msg)
								return
							}
						}
						b.StopTimer()
						srv.Close()
					},
					func(msg *arhatgopb.Msg) {}
			})

			go func() {
				time.Sleep(5 * time.Second)
				err = client.ProcessNewStream(ctrl.RefreshChannels())
				if err != nil {
					b.Log(err)
				}
			}()

			err = srv.ListenAndServe()
			if err != nil {
				b.Log(err)
			}
		})
	}
}
