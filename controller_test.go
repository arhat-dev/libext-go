package libext

import (
	"context"
	"testing"

	"arhat.dev/arhat-proto/arhatgopb"
	"arhat.dev/pkg/log"
	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/assert"
)

type testHandler struct{}

func (h *testHandler) HandleCmd(id uint64, kind arhatgopb.CmdType, payload []byte) (proto.Marshaler, error) {
	return &arhatgopb.DoneMsg{}, nil
}

func TestController_RefreshChannels(t *testing.T) {
	c, err := NewController(context.TODO(), log.NoOpLogger, &testHandler{})
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
