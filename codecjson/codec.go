package codecjson

import (
	"encoding/json"
	"io"

	"arhat.dev/libext"
)

type Codec struct{}

func (c *Codec) ContentType() string {
	return "application/json"
}

func (c *Codec) NewEncoder(w io.Writer) libext.Encoder {
	return json.NewEncoder(w)
}

func (c *Codec) NewDecoder(r io.Reader) libext.Decoder {
	return json.NewDecoder(r)
}
