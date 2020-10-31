package codecjson

import (
	"encoding/json"
	"io"

	"arhat.dev/arhat-proto/arhatgopb"

	"arhat.dev/libext"
)

type Codec struct{}

func (c *Codec) Type() arhatgopb.CodecType {
	return arhatgopb.CODEC_JSON
}

func (c *Codec) NewEncoder(w io.Writer) libext.Encoder {
	return json.NewEncoder(w)
}

func (c *Codec) NewDecoder(r io.Reader) libext.Decoder {
	return json.NewDecoder(r)
}
