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
	"testing"

	"arhat.dev/arhat-proto/arhatgopb"
	"arhat.dev/pkg/log"
	"github.com/stretchr/testify/assert"

	"arhat.dev/libext/codec"
	"arhat.dev/libext/types"

	// import default codec for test
	_ "arhat.dev/libext/codec/gogoprotobuf"
	_ "arhat.dev/libext/codec/stdjson"

	// import default network support for test
	_ "arhat.dev/pkg/nethelper/piondtls"
	_ "arhat.dev/pkg/nethelper/pipenet"
	_ "arhat.dev/pkg/nethelper/stdnet"
)

type testHandler struct{}

func (h *testHandler) SetMsgSendFunc(sendFunc types.MsgSendFunc) {}
func (h *testHandler) SendMsg(msg *arhatgopb.Msg) error          { return nil }
func (h *testHandler) HandleCmd(
	_ context.Context, _, _ uint64, _ arhatgopb.CmdType, _ []byte,
) (arhatgopb.MsgType, interface{}, error) {
	return arhatgopb.MSG_DONE, &arhatgopb.DoneMsg{}, nil
}

func TestController_RefreshChannels(t *testing.T) {
	jsonCodec, ok := codec.Get(arhatgopb.CODEC_JSON)
	if assert.True(t, ok, "no json codec found") {
		return
	}
	c, err := NewController(context.TODO(), log.NoOpLogger, jsonCodec.Marshal, &testHandler{})
	if !assert.NoError(t, err) {
		t.FailNow()
		return
	}

	assert.NoError(t, c.Start())

	cmdCh, msgCh := c.RefreshChannels()
	testRoutine(t, cmdCh, msgCh)

	newCmdCh, newMsgCh := c.RefreshChannels()

	select {
	case <-msgCh:
		close(cmdCh)
	default:
		assert.Fail(t, "msg channel not closed")
	}

	testRoutine(t, newCmdCh, newMsgCh)
}

func testRoutine(t *testing.T, cmdCh chan<- *arhatgopb.Cmd, msgCh <-chan *arhatgopb.Msg) {
	// request & reply
	cmdCh <- &arhatgopb.Cmd{
		Id:  100,
		Seq: 100,
	}
	doneMsg := <-msgCh
	assert.Equal(t, arhatgopb.MSG_DONE, doneMsg.Kind)
	assert.EqualValues(t, 100, doneMsg.Id)
	assert.EqualValues(t, 100, doneMsg.Ack)
}
