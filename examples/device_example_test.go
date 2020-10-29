package examples_test

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"arhat.dev/arhat-proto/arhatgopb"
	"arhat.dev/pkg/log"

	"arhat.dev/libext"
	"arhat.dev/libext/codecpb"
	"arhat.dev/libext/extperipheral"
)

type examplePeripheral struct{}

func (d *examplePeripheral) Operate(params map[string]string, data []byte) ([][]byte, error) {
	return [][]byte{[]byte("ok")}, nil
}

func (d *examplePeripheral) CollectMetrics(params map[string]string) ([]*arhatgopb.PeripheralMetricsMsg_Value, error) {
	return []*arhatgopb.PeripheralMetricsMsg_Value{
		{Value: 1.1, Timestamp: time.Now().UnixNano()},
	}, nil
}

func (d *examplePeripheral) Close() {}

type examplePeripheralConnector struct{}

func (c *examplePeripheralConnector) Connect(
	target string, params map[string]string, tlsConfig *arhatgopb.TLSConfig,
) (extperipheral.Peripheral, error) {
	tlsCfg, err := tlsConfig.GetTLSConfig()
	if err != nil {
		return nil, err
	}

	_ = tlsCfg
	return &examplePeripheral{}, nil
}

func ExamplePeripheralExtension() {
	appCtx := context.TODO()
	client, err := libext.NewClient(
		appCtx,
		"unix:///var/run/arhat.sock",
		&tls.Config{},
		libext.ExtensionPeripheral,
		// use protobuf codec
		&codecpb.Codec{},
	)

	if err != nil {
		panic(fmt.Errorf("failed to create client: %w", err))
	}

	ctrl, err := libext.NewController(
		appCtx,
		"my-peripheral-extension-name",
		log.Log.WithName("controller"),
		extperipheral.NewHandler(log.Log.WithName("handler"), &examplePeripheralConnector{}),
	)

	for {
		select {
		case <-appCtx.Done():
			return
		default:
			err = client.ProcessNewStream(ctrl.RefreshChannels())
			if err != nil {
				println("error happened when processing data stream", err.Error())
			}
		}
	}
}
