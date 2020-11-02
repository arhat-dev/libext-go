package util

import (
	"arhat.dev/arhat-proto/arhatgopb"

	"arhat.dev/libext/types"
)

func NewCmd(
	marshal types.MarshalFunc,
	kind arhatgopb.CmdType,
	id, seq uint64, body interface{},
) (*arhatgopb.Cmd, error) {
	payload, err := marshal(body)
	if err != nil {
		return nil, err
	}

	return &arhatgopb.Cmd{
		Kind:    kind,
		Id:      id,
		Seq:     seq,
		Payload: payload,
	}, nil
}
