package connect

import (
	"QuantumUtils/errors"
	"io"
	"net"
)

type Closeable interface {
	Close() error
}
type QTPConn interface {
	Closeable
	io.Writer
	io.Reader
}

type QTPConnAccessor struct {
	conn QTPConn
}

func (c *QTPConnAccessor) SetQTPConn(conn QTPConn) {
	c.conn = conn
}
func (c *QTPConnAccessor) GetQTPConn() QTPConn {
	return c.conn
}

// RegisterClose 注册一个流关闭服务
func RegisterClose(closeable Closeable, errorCatch func(err *errors.QError)) {
	err := closeable.Close()
	if err != nil {
		errorCatch(errors.New(err.Error()))
	}
}

type QTPReConn struct {
	conn net.Conn
	rc   *Reconnecter
}

func (c *QTPReConn) Read(b []byte) (n int, err error) {
	read, err := c.conn.Read(b)
	if err != nil && c.rc != nil {
	rer:
		conn, qErr := c.rc.TryReconnect()
		if conn == nil && qErr == nil {
			return 0, nil
		}
		if qErr != nil {
			return read, qErr
		}
		c.conn = conn
		rRead, rErr := c.conn.Read(b)
		if rErr != nil {
			goto rer
		}
		return rRead, rErr
	}
	return read, nil
}
func (c *QTPReConn) Write(b []byte) (n int, err error) {
	write, err := c.conn.Write(b)
	if err != nil && c.rc != nil {
	rew:
		conn, qErr := c.rc.TryReconnect()
		if conn == nil && qErr == nil {
			return 0, nil
		}
		if qErr != nil {
			return write, qErr
		}
		c.conn = conn
		rWrite, rErr := c.conn.Write(b)
		if rErr != nil {
			goto rew
		}
		return rWrite, rErr
	}
	return write, nil
}
func (c *QTPReConn) Close() error {
	c.rc.Close()
	return c.conn.Close()
}

func NewQTPConn(conn net.Conn, rc *Reconnecter) QTPConn {
	if rc == nil {
		return conn
	}
	return &QTPReConn{
		conn: conn,
		rc:   rc,
	}
}
