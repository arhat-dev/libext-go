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
	"bytes"
	"context"
	"fmt"
	"io"
	"testing"

	"arhat.dev/aranya-proto/aranyagopb"
	"arhat.dev/aranya-proto/aranyagopb/runtimepb"
	"arhat.dev/arhat-proto/arhatgopb"
	"arhat.dev/pkg/log"
	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/assert"

	"arhat.dev/libext/codec"
	"arhat.dev/libext/types"
)

type testRuntime struct{}

const (
	testRuntimeName          = "name"
	testRuntimeVersion       = "version"
	testRuntimeOS            = "os"
	testRuntimeOSImage       = "os-image"
	testRuntimeArch          = "arch"
	testRuntimeKernelVersion = "kernel-version"
)

func (r *testRuntime) Name() string          { return testRuntimeName }
func (r *testRuntime) Version() string       { return testRuntimeVersion }
func (r *testRuntime) OS() string            { return testRuntimeOS }
func (r *testRuntime) OSImage() string       { return testRuntimeOSImage }
func (r *testRuntime) Arch() string          { return testRuntimeArch }
func (r *testRuntime) KernelVersion() string { return testRuntimeKernelVersion }

const (
	testPodUID   = "pod"
	testImageRef = "image"
)

func (r *testRuntime) Exec(
	_ context.Context,
	podUID, container string,
	stdin io.Reader,
	stdout, stderr io.Writer,
	command []string,
	tty bool,
	errCh chan<- *aranyagopb.ErrorMsg,
) (doResize types.ResizeHandleFunc, err error) {
	if podUID != testPodUID {
		return nil, fmt.Errorf("invalid pod id for exec")
	}
	return nil, nil
}

func (r *testRuntime) Attach(
	ctx context.Context,
	podUID, container string,
	stdin io.Reader,
	stdout, stderr io.Writer,
	errCh chan<- *aranyagopb.ErrorMsg,
) (doResize types.ResizeHandleFunc, err error) {
	if podUID != testPodUID {
		return nil, fmt.Errorf("invalid pod id for attach")
	}
	return nil, nil
}

func (r *testRuntime) Logs(
	ctx context.Context,
	options *aranyagopb.LogsCmd,
	stdout, stderr io.Writer,
) error {
	if options.PodUid != testPodUID {
		return fmt.Errorf("invalid pod id for logs")
	}
	return nil
}

func (r *testRuntime) PortForward(
	ctx context.Context,
	podUID string,
	protocol string,
	port int32,
	upstream io.Reader,
	downstream io.Writer,
) error {
	if podUID != testPodUID {
		return fmt.Errorf("invalid pod id for port-forward")
	}
	return nil
}

var (
	testPodStatusResult     = &runtimepb.PodStatusMsg{Uid: testPodUID}
	testPodStatusListResult = &runtimepb.PodStatusListMsg{Pods: []*runtimepb.PodStatusMsg{testPodStatusResult}}
)

func (r *testRuntime) EnsurePod(ctx context.Context, options *runtimepb.PodEnsureCmd) (*runtimepb.PodStatusMsg, error) {
	if options.PodUid != testPodUID {
		return nil, fmt.Errorf("invalid pod id for pod ensure")
	}
	return testPodStatusResult, nil
}

func (r *testRuntime) DeletePod(ctx context.Context, options *runtimepb.PodDeleteCmd) (*runtimepb.PodStatusMsg, error) {
	if options.PodUid != testPodUID {
		return nil, fmt.Errorf("invalid pod id for pod delete")
	}
	return testPodStatusResult, nil
}

func (r *testRuntime) ListPods(ctx context.Context, options *runtimepb.PodListCmd) (*runtimepb.PodStatusListMsg, error) {
	if !options.All {
		return nil, fmt.Errorf("invalid pod list not all")
	}
	return testPodStatusListResult, nil
}

var testImageStatusListResult = &runtimepb.ImageStatusListMsg{
	Images: []*runtimepb.ImageStatusMsg{{Refs: []string{testImageRef}}},
}

func (r *testRuntime) EnsureImages(ctx context.Context, options *runtimepb.ImageEnsureCmd) (*runtimepb.ImageStatusListMsg, error) {
	if len(options.Images) == 0 ||
		options.Images[testImageRef] == nil ||
		options.Images[testImageRef].PullPolicy != runtimepb.IMAGE_PULL_IF_NOT_PRESENT {
		return nil, fmt.Errorf("invalid image names for image ensure")
	}
	return testImageStatusListResult, nil
}

func (r *testRuntime) DeleteImages(ctx context.Context, options *runtimepb.ImageDeleteCmd) (*runtimepb.ImageStatusListMsg, error) {
	if len(options.Refs) == 0 || options.Refs[0] != testImageRef {
		return nil, fmt.Errorf("invalid image delete")
	}
	return testImageStatusListResult, nil
}

func (r *testRuntime) ListImages(ctx context.Context, options *runtimepb.ImageListCmd) (*runtimepb.ImageStatusListMsg, error) {
	if len(options.Refs) == 0 || options.Refs[0] != testImageRef {
		return nil, fmt.Errorf("invalid image delete")
	}
	return testImageStatusListResult, nil
}

func TestHandler_HandleCmd(t *testing.T) {
	tests := []struct {
		name  string
		codec types.Codec
	}{
		{
			name:  "pb",
			codec: codec.GetCodec(arhatgopb.CODEC_PROTOBUF),
		},
		{
			name:  "json",
			codec: codec.GetCodec(arhatgopb.CODEC_JSON),
		},
	}
	for _, te := range tests {
		test := te
		t.Run(test.name, func(t *testing.T) {

			h := NewHandler(log.NoOpLogger, test.codec.Unmarshal, &testRuntime{})

			{
				// invalid command
				invalidPeripheralCmd, err := test.codec.Marshal(&arhatgopb.PeripheralConnectCmd{
					Target: "test",
					Params: map[string]string{"test": "test"},
					Tls:    nil,
				})
				assert.NoError(t, err)

				msg, err := h.HandleCmd(context.TODO(), 1, 0, arhatgopb.CMD_PERIPHERAL_CONNECT, invalidPeripheralCmd)
				assert.Error(t, err)
				_ = msg
			}

			testCases := []struct {
				kind       runtimepb.PacketType
				payload    proto.Marshaler
				expected   interface{}
				actualType interface{}
			}{
				{
					kind:    runtimepb.CMD_GET_INFO,
					payload: &aranyagopb.ExecOrAttachCmd{PodUid: testPodUID},
					expected: &runtimepb.RuntimeInfo{
						Name:          testRuntimeName,
						Version:       testRuntimeVersion,
						Os:            testRuntimeOS,
						OsImage:       testRuntimeOSImage,
						Arch:          testRuntimeArch,
						KernelVersion: testRuntimeKernelVersion,
					},
					actualType: &runtimepb.RuntimeInfo{},
				},
				{
					kind:     runtimepb.CMD_EXEC,
					payload:  &aranyagopb.ExecOrAttachCmd{PodUid: testPodUID},
					expected: nil,
				},
				{
					kind:     runtimepb.CMD_ATTACH,
					payload:  &aranyagopb.ExecOrAttachCmd{PodUid: testPodUID},
					expected: nil,
				},
				{
					kind:     runtimepb.CMD_LOGS,
					payload:  &aranyagopb.LogsCmd{PodUid: testPodUID},
					expected: nil,
				},
				{
					kind:     runtimepb.CMD_PORT_FORWARD,
					payload:  &aranyagopb.PortForwardCmd{PodUid: testPodUID},
					expected: nil,
				},
				{
					kind:       runtimepb.CMD_POD_LIST,
					payload:    &runtimepb.PodListCmd{All: true},
					expected:   testPodStatusListResult,
					actualType: new(runtimepb.PodStatusListMsg),
				},
				{
					kind:       runtimepb.CMD_POD_DELETE,
					payload:    &runtimepb.PodDeleteCmd{PodUid: testPodUID},
					expected:   testPodStatusResult,
					actualType: new(runtimepb.PodStatusMsg),
				},
				{
					kind:       runtimepb.CMD_POD_ENSURE,
					payload:    &runtimepb.PodEnsureCmd{PodUid: testPodUID},
					expected:   testPodStatusResult,
					actualType: new(runtimepb.PodStatusMsg),
				},
				{
					kind:       runtimepb.CMD_IMAGE_LIST,
					payload:    &runtimepb.ImageListCmd{Refs: []string{testImageRef}},
					expected:   testImageStatusListResult,
					actualType: new(runtimepb.ImageStatusListMsg),
				},
				{
					kind:       runtimepb.CMD_IMAGE_DELETE,
					payload:    &runtimepb.ImageDeleteCmd{Refs: []string{testImageRef}},
					expected:   testImageStatusListResult,
					actualType: new(runtimepb.ImageStatusListMsg),
				},
				{
					kind: runtimepb.CMD_IMAGE_ENSURE,
					payload: &runtimepb.ImageEnsureCmd{Images: map[string]*runtimepb.ImagePullSpec{testImageRef: {
						AuthConfig: nil,
						PullPolicy: runtimepb.IMAGE_PULL_IF_NOT_PRESENT,
					}}},
					expected:   testImageStatusListResult,
					actualType: new(runtimepb.ImageStatusListMsg),
				},
			}

			for _, ca := range testCases {
				c := ca
				t.Run(c.kind.String(), func(t *testing.T) {
					h := NewHandler(log.NoOpLogger, test.codec.Unmarshal, &testRuntime{})

					resultCh := make(chan []byte)

					encodeRuntimePkt := func(kind runtimepb.PacketType, pkt proto.Marshaler) []byte {
						data, err := pkt.Marshal()
						if !assert.NoError(t, err) {
							panic(err)
						}

						data, err = codec.GetCodec(arhatgopb.CODEC_PROTOBUF).Marshal(&runtimepb.Packet{
							Kind:    kind,
							Payload: data,
						})
						if !assert.NoError(t, err) {
							panic(err)
						}

						return data
					}

					h.SetMsgSendFunc(func(msg *arhatgopb.Msg) error {
						if c.expected == nil {
							return fmt.Errorf("unexpected nil message sent")
						}
						assert.NotNil(t, msg)

						buf := new(bytes.Buffer)
						assert.NoError(t, test.codec.NewEncoder(buf).Encode(msg))

						resultCh <- buf.Bytes()

						return nil
					})

					ret, err := h.HandleCmd(
						context.TODO(), 1, 1,
						arhatgopb.CMD_RUNTIME_ARANYA_PROTO,
						encodeRuntimePkt(c.kind, c.payload),
					)

					if !assert.NoError(t, err) {
						return
					}

					// all actions are async
					assert.Nil(t, ret)

					if c.expected == nil {
						return
					}

					data := <-resultCh

					actualRet := new(arhatgopb.Msg)

					assert.NoError(t, test.codec.NewDecoder(bytes.NewReader(data)).Decode(actualRet))
					assert.EqualValues(t, arhatgopb.MSG_RUNTIME_ARANYA_PROTO, actualRet.Kind)

					pkt := new(runtimepb.Packet)
					assert.NoError(t, codec.GetCodec(arhatgopb.CODEC_PROTOBUF).Unmarshal(actualRet.Payload, pkt))
					assert.NoError(t, codec.GetCodec(arhatgopb.CODEC_PROTOBUF).Unmarshal(pkt.Payload, c.actualType))

					assert.EqualValues(t, c.expected, c.actualType)
				})
			}
		})
	}
}
