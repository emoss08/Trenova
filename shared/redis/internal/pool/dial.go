package pool

import (
	"context"
	"crypto/tls"
	"net"
	"time"

	"github.com/emoss08/trenova/shared/redis/internal/resp"
)

const (
	tcpKeepAlive   = 5 * time.Minute
	readBufferSize = 32 << 10
)

type DialOptions struct {
	Addr        string
	Username    string
	Password    string
	DB          int
	DialTimeout time.Duration
	TLSConfig   *tls.Config
	Wrap        func(ctx context.Context, dial func(context.Context) (net.Conn, error)) (net.Conn, error)
}

func Connector(o *DialOptions) func(ctx context.Context) (*Conn, error) {
	return func(ctx context.Context) (*Conn, error) {
		var (
			nc  net.Conn
			err error
		)
		if o.Wrap != nil {
			nc, err = o.Wrap(ctx, o.dial)
		} else {
			nc, err = o.dial(ctx)
		}
		if err != nil {
			return nil, err
		}

		cn := NewConn(nc, readBufferSize)
		if err = o.handshake(cn); err != nil {
			_ = cn.Close()
			return nil, err
		}
		return cn, nil
	}
}

func (o *DialOptions) dial(ctx context.Context) (net.Conn, error) {
	d := net.Dialer{Timeout: o.DialTimeout, KeepAlive: tcpKeepAlive}
	nc, err := d.DialContext(ctx, "tcp", o.Addr)
	if err != nil || o.TLSConfig == nil {
		return nc, err
	}

	cfg := o.TLSConfig.Clone()
	if cfg.ServerName == "" {
		if host, _, splitErr := net.SplitHostPort(o.Addr); splitErr == nil {
			cfg.ServerName = host
		}
	}
	tc := tls.Client(nc, cfg)
	hctx, cancel := context.WithTimeout(ctx, o.DialTimeout)
	defer cancel()
	if err = tc.HandshakeContext(hctx); err != nil {
		_ = nc.Close()
		return nil, err
	}
	return tc, nil
}

func (o *DialOptions) handshake(cn *Conn) error {
	cmds := make([][]any, 0, 2)
	if o.Password != "" {
		if o.Username != "" {
			cmds = append(cmds, []any{"auth", o.Username, o.Password})
		} else {
			cmds = append(cmds, []any{"auth", o.Password})
		}
	}
	if o.DB > 0 {
		cmds = append(cmds, []any{"select", o.DB})
	}
	if len(cmds) == 0 {
		return nil
	}

	cn.Reset()
	for _, args := range cmds {
		if err := cn.Append(args); err != nil {
			return err
		}
	}
	deadline := time.Now().Add(o.DialTimeout)
	if err := cn.Flush(deadline); err != nil {
		return err
	}
	var firstErr error
	for range cmds {
		v, err := cn.Read(deadline)
		if err != nil {
			return err
		}
		if v.Kind == resp.KindError && firstErr == nil {
			firstErr = resp.Error(v.Str)
		}
	}
	return firstErr
}
