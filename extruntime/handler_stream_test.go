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

package extruntime

import (
	"context"
	"io"
	"net"
	"testing"
	"time"

	"arhat.dev/aranya-proto/aranyagopb"
	"arhat.dev/aranya-proto/aranyagopb/runtimepb"
	"arhat.dev/arhat-proto/arhatgopb"
	"arhat.dev/pkg/log"
	"arhat.dev/pkg/wellknownerrors"
	"github.com/stretchr/testify/assert"

	"arhat.dev/libext/types"

	// import default codec for test
	_ "arhat.dev/libext/codec/codecjson"
	_ "arhat.dev/libext/codec/codecpb"
)

const (
	testStreamData = "data"
)

type testStreamRuntime struct {
	t *testing.T
}

func (r *testStreamRuntime) Name() string          { return "" }
func (r *testStreamRuntime) Version() string       { return "" }
func (r *testStreamRuntime) OS() string            { return "" }
func (r *testStreamRuntime) OSImage() string       { return "" }
func (r *testStreamRuntime) Arch() string          { return "" }
func (r *testStreamRuntime) KernelVersion() string { return "" }

func (r *testStreamRuntime) Exec(
	_ context.Context,
	podUID, container string,
	stdin io.Reader,
	stdout, stderr io.Writer,
	command []string,
	tty bool,
) (
	doResize types.ResizeHandleFunc,
	_ <-chan *aranyagopb.ErrorMsg,
	err error,
) {
	if !assert.NotNil(r.t, stdin, "no stdin provided") {
		return nil, nil, wellknownerrors.ErrNotSupported
	}

	if !assert.NotNil(r.t, stdout, "no stdout provided") {
		return nil, nil, wellknownerrors.ErrNotSupported
	}

	if !assert.NotNil(r.t, stderr, "no stderr provided") {
		return nil, nil, wellknownerrors.ErrNotSupported
	}

	assert.EqualValues(r.t, []string{"test"}, command)

	errCh := make(chan *aranyagopb.ErrorMsg)
	go func() {
		defer close(errCh)
		buf := make([]byte, len(testStreamData))
		_, err = stdin.Read(buf)
		assert.NoError(r.t, err, "failed to read from stdin")
		assert.EqualValues(r.t, testStreamData, string(buf))

		_, err = stderr.Write([]byte(testStreamData))
		assert.NoError(r.t, err, "failed to write stderr data")

		_, err = stdout.Write([]byte(testStreamData))
		assert.NoError(r.t, err, "failed to write stdout data")
	}()

	return nil, errCh, nil
}

func (r *testStreamRuntime) Attach(
	ctx context.Context,
	podUID, container string,
	stdin io.Reader,
	stdout, stderr io.Writer,
) (
	doResize types.ResizeHandleFunc,
	_ <-chan *aranyagopb.ErrorMsg,
	err error,
) {
	if !assert.NotNil(r.t, stdin, "no stdin provided") {
		return nil, nil, wellknownerrors.ErrNotSupported
	}

	if !assert.NotNil(r.t, stdout, "no stdout provided") {
		return nil, nil, wellknownerrors.ErrNotSupported
	}

	if !assert.NotNil(r.t, stderr, "no stderr provided") {
		return nil, nil, wellknownerrors.ErrNotSupported
	}

	errCh := make(chan *aranyagopb.ErrorMsg)
	go func() {
		defer close(errCh)

		buf := make([]byte, len(testStreamData))
		_, err = stdin.Read(buf)
		assert.NoError(r.t, err, "failed to read from stdin")
		assert.EqualValues(r.t, testStreamData, string(buf))

		_, err = stderr.Write([]byte(testStreamData))
		assert.NoError(r.t, err, "failed to write stderr data")

		_, err = stdout.Write([]byte(testStreamData))
		assert.NoError(r.t, err, "failed to write stdout data")
	}()

	return nil, errCh, nil
}

func (r *testStreamRuntime) Logs(
	ctx context.Context,
	options *aranyagopb.LogsCmd,
	stdout, stderr io.Writer,
) error {
	if !assert.NotNil(r.t, stdout, "no stdout provided") {
		return wellknownerrors.ErrNotSupported
	}

	if !assert.NotNil(r.t, stderr, "no stderr provided") {
		return wellknownerrors.ErrNotSupported
	}

	_, err := stderr.Write([]byte(testStreamData))
	assert.NoError(r.t, err, "failed to write stderr data")

	_, err = stdout.Write([]byte(testStreamData))
	assert.NoError(r.t, err, "failed to write stdout data")

	return nil
}

func (r *testStreamRuntime) PortForward(
	ctx context.Context,
	podUID string,
	protocol string,
	port int32,
	upstream io.Reader,
) (
	downstream io.ReadCloser,
	closeWriter func(),
	readErrCh <-chan error,
	err error,
) {
	if !assert.NotNil(r.t, upstream, "no upstream provided") {
		return nil, nil, nil, wellknownerrors.ErrNotSupported
	}

	assert.EqualValues(r.t, 12345, port)

	pr, pw := net.Pipe()

	errCh := make(chan error)
	go func() {
		defer close(errCh)

		buf := make([]byte, len(testStreamData))
		_, err := upstream.Read(buf)
		assert.NoError(r.t, err, "failed to read from upstream")
		assert.EqualValues(r.t, testStreamData, string(buf))

		_, err = pw.Write([]byte(testStreamData))
		assert.NoError(r.t, err, "failed to write downstream data")
	}()

	return pr, func() { _ = pw.Close() }, errCh, nil
}

func (r *testStreamRuntime) EnsurePod(ctx context.Context, options *runtimepb.PodEnsureCmd) (*runtimepb.PodStatusMsg, error) {
	return nil, wellknownerrors.ErrNotSupported
}

func (r *testStreamRuntime) DeletePod(ctx context.Context, options *runtimepb.PodDeleteCmd) (*runtimepb.PodStatusMsg, error) {
	return nil, wellknownerrors.ErrNotSupported
}

func (r *testStreamRuntime) ListPods(ctx context.Context, options *runtimepb.PodListCmd) (*runtimepb.PodStatusListMsg, error) {
	return nil, wellknownerrors.ErrNotSupported
}

func (r *testStreamRuntime) EnsureImages(ctx context.Context, options *runtimepb.ImageEnsureCmd) (*runtimepb.ImageStatusListMsg, error) {
	return nil, wellknownerrors.ErrNotSupported
}

func (r *testStreamRuntime) DeleteImages(ctx context.Context, options *runtimepb.ImageDeleteCmd) (*runtimepb.ImageStatusListMsg, error) {
	return nil, wellknownerrors.ErrNotSupported
}

func (r *testStreamRuntime) ListImages(ctx context.Context, options *runtimepb.ImageListCmd) (*runtimepb.ImageStatusListMsg, error) {
	return nil, wellknownerrors.ErrNotSupported
}

func TestHandler_Stream(t *testing.T) {
	for _, pkt := range []*runtimepb.Packet{
		{
			Kind: runtimepb.CMD_EXEC,
			Payload: func() []byte {
				data, err := (&aranyagopb.ExecOrAttachCmd{
					Stdin:   true,
					Stdout:  true,
					Stderr:  true,
					Tty:     true,
					Command: []string{"test"},
					Envs:    nil,
				}).Marshal()

				assert.NoError(t, err)
				return data
			}(),
		},
		{
			Kind: runtimepb.CMD_ATTACH,
			Payload: func() []byte {
				data, err := (&aranyagopb.ExecOrAttachCmd{
					PodUid:    "",
					Container: "",
					Stdin:     true,
					Stdout:    true,
					Stderr:    true,
					Tty:       true,
				}).Marshal()

				assert.NoError(t, err)
				return data
			}(),
		},
		{
			Kind: runtimepb.CMD_LOGS,
			Payload: func() []byte {
				data, err := (&aranyagopb.LogsCmd{
					PodUid:     "",
					Container:  "",
					Follow:     false,
					Timestamp:  false,
					Since:      "",
					TailLines:  0,
					BytesLimit: 0,
					Previous:   false,
					Path:       "",
				}).Marshal()

				assert.NoError(t, err)
				return data
			}(),
		},
		{
			Kind: runtimepb.CMD_PORT_FORWARD,
			Payload: func() []byte {
				data, err := (&aranyagopb.PortForwardCmd{
					PodUid:   "",
					Protocol: "tcp",
					Port:     12345,
				}).Marshal()

				assert.NoError(t, err)
				return data
			}(),
		},
	} {
		t.Run(pkt.Kind.String(), func(t *testing.T) {
			pktBytes, err := pkt.Marshal()
			if !assert.NoError(t, err) {
				return
			}

			h := NewHandler(log.NoOpLogger, 4096, &testStreamRuntime{t: t})

			stdoutCh := make(chan struct{})
			stderrCh := make(chan struct{})

			count := 0
			h.SetMsgSendFunc(func(msg *arhatgopb.Msg) error {
				switch msg.Kind {
				case arhatgopb.MSG_DATA_OUTPUT:
					count++
					switch pkt.Kind {
					case runtimepb.CMD_LOGS, runtimepb.CMD_PORT_FORWARD:
						if count != 1 {
							return nil
						}
					}

					close(stdoutCh)
				case arhatgopb.MSG_RUNTIME_DATA_STDERR:
					close(stderrCh)
				default:
					assert.Failf(t, "unexpected message type %q", msg.Kind.String())
				}

				return nil
			})

			_, ret, err := h.HandleCmd(context.TODO(), 1, 0, arhatgopb.CMD_RUNTIME_ARANYA_PROTO, pktBytes)
			assert.Nil(t, ret)
			assert.NoError(t, err)

			time.Sleep(time.Second)
			_, ret, err = h.HandleCmd(context.TODO(), 1, 0, arhatgopb.CMD_DATA_INPUT, []byte(testStreamData))
			assert.Nil(t, ret)
			assert.NoError(t, err)

			<-stdoutCh
			if pkt.Kind != runtimepb.CMD_PORT_FORWARD {
				<-stdoutCh
			}
		})
	}
}
