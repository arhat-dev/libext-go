package examples_test

import (
	"arhat.dev/arhat-proto/arhatgopb"
	"arhat.dev/libext"
	"arhat.dev/libext/codecpb"
	"arhat.dev/libext/extdevice"
	"arhat.dev/pkg/log"
	"context"
	"crypto/tls"
	"fmt"
	"time"
)

type exampleDevice struct{}

func (d *exampleDevice) Operate(params map[string]string, data []byte) ([][]byte, error) {
	return [][]byte{[]byte("ok")}, nil
}

func (d *exampleDevice) CollectMetrics(params map[string]string) ([]*arhatgopb.DeviceMetricsMsg_Value, error) {
	return []*arhatgopb.DeviceMetricsMsg_Value{
		{Value: 1.1, Timestamp: time.Now().UnixNano()},
	}, nil
}

func (d *exampleDevice) Close() {}

type exampleDeviceConnector struct{}

func (c *exampleDeviceConnector) Connect(
	target string, params map[string]string, tlsConfig *arhatgopb.TLSConfig,
) (extdevice.Device, error) {
	tlsCfg, err := tlsConfig.GetTLSConfig()
	if err != nil {
		return nil, err
	}

	_ = tlsCfg
	return &exampleDevice{}, nil
}

func ExampleDeviceExtension() {
	appCtx := context.TODO()
	client, err := libext.NewClient(
		appCtx,
		"unix:///var/run/arhat.sock",
		&tls.Config{},
		libext.ExtensionDevice,
		// use protobuf codec
		&codecpb.Codec{},
	)

	if err != nil {
		panic(fmt.Errorf("failed to create client: %w", err))
	}

	ctrl, err := libext.NewController(
		appCtx,
		"my-device-extension-name",
		log.Log.WithName("controller"),
		extdevice.NewHandler(log.Log.WithName("handler"), &exampleDeviceConnector{}),
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
