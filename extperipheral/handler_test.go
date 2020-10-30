package extperipheral

import (
	"testing"
	"time"

	"arhat.dev/arhat-proto/arhatgopb"
	"arhat.dev/pkg/log"
	"github.com/stretchr/testify/assert"
)

type testPeripheral struct {
}

func (p *testPeripheral) Operate(params map[string]string, data []byte) ([][]byte, error) {
	return [][]byte{[]byte("test")}, nil
}

func (p *testPeripheral) CollectMetrics(params map[string]string) ([]*arhatgopb.PeripheralMetricsMsg_Value, error) {
	return []*arhatgopb.PeripheralMetricsMsg_Value{
		{Value: 1, Timestamp: time.Unix(0, 0).UnixNano()},
	}, nil
}

func (p *testPeripheral) Close() {}

type testPeripheralConnector struct {
}

func (p *testPeripheralConnector) Connect(target string, params map[string]string, tlsConfig *arhatgopb.TLSConfig) (Peripheral, error) {
	return &testPeripheral{}, nil
}

func TestHandler_HandleCmd(t *testing.T) {
	h := NewHandler(log.NoOpLogger, &testPeripheralConnector{})

	{
		connectCmdBytes, err := (&arhatgopb.PeripheralConnectCmd{
			Target: "test",
			Params: map[string]string{"test": "test"},
			Tls:    nil,
		}).Marshal()
		assert.NoError(t, err)

		msg, err := h.HandleCmd(1, arhatgopb.CMD_PERIPHERAL_CONNECT, connectCmdBytes)
		assert.NoError(t, err)
		assert.IsType(t, &arhatgopb.DoneMsg{}, msg)
	}

	{
		operateCmdBytes, err := (&arhatgopb.PeripheralOperateCmd{
			Params: map[string]string{"test": "test"},
			Data:   nil,
		}).Marshal()
		assert.NoError(t, err)

		msg, err := h.HandleCmd(1, arhatgopb.CMD_PERIPHERAL_OPERATE, operateCmdBytes)
		assert.NoError(t, err)
		assert.IsType(t, &arhatgopb.PeripheralOperationResultMsg{}, msg)
	}

	{
		metricCmdBytes, err := (&arhatgopb.PeripheralMetricsCollectCmd{
			Params: map[string]string{"test": "test"},
		}).Marshal()
		assert.NoError(t, err)

		msg, err := h.HandleCmd(1, arhatgopb.CMD_PERIPHERAL_COLLECT_METRICS, metricCmdBytes)
		assert.NoError(t, err)
		assert.IsType(t, &arhatgopb.PeripheralMetricsMsg{}, msg)
	}

	{
		closeCmdBytes, err := (&arhatgopb.PeripheralCloseCmd{}).Marshal()
		assert.NoError(t, err)

		msg, err := h.HandleCmd(1, arhatgopb.CMD_PERIPHERAL_CLOSE, closeCmdBytes)
		assert.NoError(t, err)
		assert.IsType(t, &arhatgopb.DoneMsg{}, msg)
	}

	{
		invalidOperateCmdBytes, err := (&arhatgopb.PeripheralOperateCmd{
			Params: map[string]string{"test": "test"},
		}).Marshal()
		assert.NoError(t, err)

		_, err = h.HandleCmd(1, arhatgopb.CMD_PERIPHERAL_OPERATE, invalidOperateCmdBytes)
		assert.Error(t, err)
	}
}
