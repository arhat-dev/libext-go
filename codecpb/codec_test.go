package codecpb

import (
	"bytes"
	"encoding/binary"
	"reflect"
	"testing"

	"arhat.dev/arhat-proto/arhatgopb"
	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/assert"
)

func TestCodec(t *testing.T) {
	type testCase struct {
		name      string
		msg       proto.Message
		dataBytes []byte
	}

	msgs := []proto.Message{
		&arhatgopb.Cmd{
			Kind:    1,
			Id:      1,
			Seq:     1,
			Payload: []byte("test"),
		},
		&arhatgopb.Msg{
			Kind:    1,
			Id:      1,
			Ack:     1,
			Payload: []byte("test"),
		},
	}

	var tests []testCase
	for i, m := range msgs {
		data, err := proto.Marshal(m)
		if !assert.NoError(t, err) {
			assert.FailNow(t, "failed to unmarshal %d msg", i)
			return
		}

		buf := make([]byte, 10)
		buf = buf[:binary.PutUvarint(buf, uint64(len(data)))]

		tests = append(tests, testCase{
			name:      reflect.TypeOf(m).String(),
			msg:       msgs[i],
			dataBytes: append(buf, data...),
		})
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := new(Codec)
			buf := new(bytes.Buffer)
			enc := c.NewEncoder(buf)
			assert.NoError(t, enc.Encode(test.msg))
			assert.EqualValues(t, test.dataBytes, buf.Bytes())

			dec := c.NewDecoder(bytes.NewReader(test.dataBytes))

			m := proto.Clone(test.msg)
			m.Reset()
			assert.NoError(t, dec.Decode(m))
			assert.EqualValues(t, test.msg, m)
		})
	}
}
