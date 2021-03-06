// +build !nocodec_stdjson

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

package stdjson

import (
	"encoding/json"
	"io"

	"arhat.dev/arhat-proto/arhatgopb"

	"arhat.dev/libext/codec"
)

func init() {
	codec.Register(arhatgopb.CODEC_JSON, new(Codec))
}

type Codec struct{}

func (c *Codec) Type() arhatgopb.CodecType {
	return arhatgopb.CODEC_JSON
}

func (c *Codec) NewEncoder(w io.Writer) codec.Encoder {
	return json.NewEncoder(w)
}

func (c *Codec) NewDecoder(r io.Reader) codec.Decoder {
	return json.NewDecoder(r)
}

func (c *Codec) Marshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

func (c *Codec) Unmarshal(data []byte, out interface{}) error {
	return json.Unmarshal(data, out)
}
