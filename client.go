package libext

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/url"
	"strings"
	"time"

	"arhat.dev/arhat-proto/arhatgopb"
	"github.com/pion/dtls/v2"
	"golang.org/x/sync/errgroup"
)

type (
	connectFunc func() (net.Conn, error)
)

func NewClient(
	ctx context.Context,
	kind arhatgopb.ExtensionType,
	name string,
	codec Codec,

	// connection management
	dialer *net.Dialer,
	endpointURL string,
	tlsConfig *tls.Config,
) (*Client, error) {
	u, err := url.Parse(endpointURL)
	if err != nil {
		return nil, fmt.Errorf("invalid endpoint url: %w", err)
	}

	reg := &arhatgopb.RegisterMsg{
		Name:          name,
		ExtensionType: kind,
		Codec:         codec.Type(),
	}
	regMsg, err := arhatgopb.NewMsg(0, 0, reg)
	if err != nil {
		return nil, fmt.Errorf("failed to create register message: %w", err)
	}

	regMsgBytes, err := json.Marshal(regMsg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal register message: %w", err)
	}

	if dialer == nil {
		dialer = &net.Dialer{
			Timeout:       0,
			Deadline:      time.Time{},
			LocalAddr:     nil,
			FallbackDelay: 0,
			KeepAlive:     0,
			Resolver:      nil,
			Control:       nil,
		}
	}

	var (
		connector connectFunc
	)
	switch s := strings.ToLower(u.Scheme); s {
	case "tcp", "tcp4", "tcp6": // nolint:goconst
		_, err = net.ResolveTCPAddr(s, u.Host)
		connector = func() (net.Conn, error) {
			return dialer.DialContext(ctx, s, u.Host)
		}
	case "udp", "udp4", "udp6": // nolint:goconst
		_, err = net.ResolveUDPAddr(s, u.Host)
		connector = func() (net.Conn, error) {
			return dialer.DialContext(ctx, s, u.Host)
		}
	case "unix", "unixgram": // nolint:goconst
		_, err = net.ResolveUnixAddr(s, u.Path)
		connector = func() (net.Conn, error) {
			return dialer.DialContext(ctx, s, u.Path)
		}
	//case "fifo":
	//	connector = func() (net.Conn, error) {
	//		return nil, err
	//	}
	default:
		return nil, fmt.Errorf("unsupported endpoint scheme %s", u.Scheme)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to resolve %s address: %w", u.Scheme, err)
	}

	return &Client{
		ctx: ctx,

		codec:  codec,
		regMsg: regMsgBytes,

		createConnection: func() (conn net.Conn, err error) {
			conn, err = connector()
			if err != nil {
				return nil, err
			}
			if tlsConfig == nil {
				return conn, nil
			}

			defer func() {
				if err != nil {
					_ = conn.Close()
				}
			}()

			_, isPktConn := conn.(net.PacketConn)
			if isPktConn {
				var cs []dtls.CipherSuiteID
				for i := range tlsConfig.CipherSuites {
					cs = append(cs, dtls.CipherSuiteID(tlsConfig.CipherSuites[i]))
				}
				return dtls.ClientWithContext(ctx, conn, &dtls.Config{
					Certificates:          tlsConfig.Certificates,
					CipherSuites:          cs,
					InsecureSkipVerify:    tlsConfig.InsecureSkipVerify,
					VerifyPeerCertificate: tlsConfig.VerifyPeerCertificate,
					RootCAs:               tlsConfig.RootCAs,
					ClientCAs:             tlsConfig.ClientCAs,
					ServerName:            tlsConfig.ServerName,
					ConnectContextMaker: func() (context.Context, func()) {
						return context.WithCancel(ctx)
					},

					// TODO: support more dTLS options
					SignatureSchemes:       nil,
					SRTPProtectionProfiles: nil,
					ClientAuth:             0,
					ExtendedMasterSecret:   0,
					FlightInterval:         0,
					PSK:                    nil,
					PSKIdentityHint:        nil,
					InsecureHashes:         false,
					LoggerFactory:          nil,
					MTU:                    0,
					ReplayProtectionWindow: 0,
				})
			}

			return tls.Client(conn, tlsConfig), nil
		},
	}, nil
}

type Client struct {
	ctx context.Context

	codec  Codec
	regMsg []byte

	createConnection connectFunc
}

// ProcessNewStream creates a new connection and handles message stream until connection lost
// or msgCh closed
// the provided `cmdCh` and `msgCh` are expected to be freshly created
// usually this function is used in conjunction with Controller.RefreshChannels
func (c *Client) ProcessNewStream(
	cmdCh chan<- *arhatgopb.Cmd,
	msgCh <-chan *arhatgopb.Msg,
) error {
	conn, err := c.createConnection()
	if err != nil {
		return fmt.Errorf("failed to dial endpoint: %w", err)
	}

	defer func() {
		_ = conn.Close()
	}()

	_, err = conn.Write(c.regMsg)
	if err != nil {
		return fmt.Errorf("failed to register myself: %w", err)
	}

	wg, ctx := errgroup.WithContext(c.ctx)

	wg.Go(func() error {
		enc := c.codec.NewEncoder(conn)

		defer func() {
			_ = conn.Close()
		}()

		for msg := range msgCh {
			err2 := enc.Encode(msg)
			if err2 != nil {
				return fmt.Errorf("failed to marshal and send msg: %w", err2)
			}
		}

		return io.EOF
	})

	wg.Go(func() error {
		defer func() {
			close(cmdCh)
		}()

		dec := c.codec.NewDecoder(conn)
		for {
			cmd := new(arhatgopb.Cmd)
			err2 := checkNetworkReadErr(dec.Decode(cmd))
			if err2 != nil {
				return err2
			}

			select {
			case cmdCh <- cmd:
			case <-ctx.Done():
				return nil
			}
		}
	})

	err = wg.Wait()
	if err != nil && err != io.EOF {
		return err
	}

	return nil
}

func checkNetworkReadErr(err error) error {
	if err == nil {
		return nil
	}

	switch t := err.(type) {
	case *net.OpError:
		if t.Err.Error() == "use of closed network connection" {
			return io.EOF
		}
	default:
		return t
	}

	return err
}
