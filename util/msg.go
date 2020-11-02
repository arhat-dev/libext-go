package util

import (
	"arhat.dev/arhat-proto/arhatgopb"

	"arhat.dev/libext/types"
)

func NewMsg(
	marshal types.MarshalFunc,
	kind arhatgopb.MsgType,
	id, ack uint64, body interface{},
) (*arhatgopb.Msg, error) {
	payload, err := marshal(body)
	if err != nil {
		return nil, err
	}

	return &arhatgopb.Msg{
		Kind:    kind,
		Id:      id,
		Ack:     ack,
		Payload: payload,
	}, nil
}

func GetMsgType(m interface{}) arhatgopb.MsgType {
	switch m.(type) {
	case *arhatgopb.RegisterMsg:
		return arhatgopb.MSG_REGISTER
	case *arhatgopb.PeripheralOperationResultMsg:
		return arhatgopb.MSG_PERIPHERAL_OPERATION_RESULT
	case *arhatgopb.PeripheralMetricsMsg:
		return arhatgopb.MSG_PERIPHERAL_METRICS
	case *arhatgopb.DoneMsg:
		return arhatgopb.MSG_DONE
	case *arhatgopb.ErrorMsg:
		return arhatgopb.MSG_ERROR
	case *arhatgopb.PeripheralEventMsg:
		return arhatgopb.MSG_PERIPHERAL_EVENTS
	default:
		return 0
	}
}
