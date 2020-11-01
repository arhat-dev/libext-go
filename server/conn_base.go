package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"sync"

	"arhat.dev/arhat-proto/arhatgopb"
	"arhat.dev/pkg/log"

	"arhat.dev/libext/codecjson"
	"arhat.dev/libext/codecpb"
	"arhat.dev/libext/types"
)

func newBaseConnectionManager(
	ctx context.Context,
	logger log.Interface,
	addr net.Addr,
	handleFunc netConnectionHandleFunc,
) *baseConnectionManager {
	return &baseConnectionManager{
		ctx:           ctx,
		logger:        logger,
		addr:          addr,
		handleNewConn: handleFunc,
		mu:            new(sync.RWMutex),
	}
}

type baseConnectionManager struct {
	ctx    context.Context
	logger log.Interface
	addr   net.Addr

	handleNewConn netConnectionHandleFunc

	mu *sync.RWMutex
}

func (m *baseConnectionManager) validateConnection(
	conn io.Reader,
) (_ arhatgopb.ExtensionType, _ string, _ types.Codec, err error) {
	onceDec := json.NewDecoder(conn)
	onceDec.DisallowUnknownFields()

	initialMsg := new(arhatgopb.Msg)
	err = onceDec.Decode(initialMsg)
	if err != nil {
		return 0, "", nil, fmt.Errorf("invalid initial message: %w", err)
	}

	if initialMsg.Kind != arhatgopb.MSG_REGISTER {
		return 0, "", nil, fmt.Errorf("initial message is not register message")
	}

	regMsg := new(arhatgopb.RegisterMsg)
	err = regMsg.Unmarshal(initialMsg.Payload)
	if err != nil {
		return 0, "", nil, fmt.Errorf("failed to unmarshal register message: %w", err)
	}

	var codec types.Codec
	switch regMsg.Codec {
	case arhatgopb.CODEC_PROTOBUF:
		codec = new(codecpb.Codec)
	case arhatgopb.CODEC_JSON:
		codec = new(codecjson.Codec)
	default:
		return 0, "", nil, fmt.Errorf("unsupported codec")
	}

	return regMsg.ExtensionType, regMsg.Name, codec, nil
}
