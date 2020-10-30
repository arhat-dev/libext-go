package libext_test

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"

	"arhat.dev/arhat-proto/arhatgopb"
	"github.com/stretchr/testify/assert"

	"arhat.dev/libext"
	"arhat.dev/libext/codecjson"
)

func TestClient_ProcessNewStream(t *testing.T) {
	l, err := net.Listen("tcp", "localhost:0")
	if !assert.NoError(t, err) {
		t.FailNow()
		return
	}

	defer func() {
		_ = l.Close()
	}()

	go func() {
		_ = http.Serve(l, http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			assert.Equal(t, http.MethodPost, request.Method)
			writer.Header().Set("Transfer-Encoding", "chunked")

			writer.WriteHeader(http.StatusOK)

			_, _ = writer.Write([]byte("{}"))

			time.Sleep(time.Second)
		}))
	}()

	client, err := libext.NewClient(context.TODO(), "http://"+l.Addr().String(), nil, "/test", new(codecjson.Codec))
	if !assert.NoError(t, err) {
		t.FailNow()
		return
	}

	cmdCh, msgCh := make(chan *arhatgopb.Cmd), make(chan *arhatgopb.Msg)

	go close(msgCh)

	assert.NoError(t, client.ProcessNewStream(cmdCh, msgCh))
	select {
	case <-cmdCh:
	default:
		assert.FailNow(t, "cmdCh not closed")
	}
}
