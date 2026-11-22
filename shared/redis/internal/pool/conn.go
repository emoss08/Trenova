package pool

import (
	"net"
	"time"

	"github.com/emoss08/trenova/shared/redis/internal/resp"
)

const maxRetainedBuffer = 1 << 20

type Conn struct {
	nc  net.Conn
	rd  *resp.Reader
	buf []byte
	res any
	rdl time.Time
}

func NewConn(nc net.Conn, readBufferSize int) *Conn {
	return &Conn{nc: nc, rd: resp.NewReader(nc, readBufferSize)}
}

func (c *Conn) Reset() {
	c.buf = c.buf[:0]
}

func (c *Conn) Append(args []any) error {
	buf, err := resp.AppendCommand(c.buf, args)
	c.buf = buf
	return err
}

func (c *Conn) Flush(deadline time.Time) error {
	if err := c.nc.SetWriteDeadline(deadline); err != nil {
		return err
	}
	_, err := c.nc.Write(c.buf)
	if cap(c.buf) > maxRetainedBuffer {
		c.buf = nil
	} else {
		c.buf = c.buf[:0]
	}
	return err
}

func (c *Conn) Read(deadline time.Time) (resp.Value, error) {
	if !deadline.Equal(c.rdl) {
		if err := c.nc.SetReadDeadline(deadline); err != nil {
			return resp.Value{}, err
		}
		c.rdl = deadline
	}
	return c.rd.Read()
}

func (c *Conn) Interrupt() {
	_ = c.nc.SetDeadline(time.Now())
}

func (c *Conn) Abort() {
	_ = c.nc.Close()
}

func (c *Conn) Close() error {
	return c.nc.Close()
}
