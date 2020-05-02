package memguarded

import (
	"github.com/n0rad/go-erlog/data"
	"github.com/n0rad/go-erlog/errs"
	"net"
	"time"
)

type Client struct {
	SocketPath     string
	SocketPassword *Service

	conn *net.Conn
}

func (c *Client) Connect() error {
	conn, err := net.Dial("unix", c.SocketPath)
	if err != nil {
		return errs.WithEF(err, data.WithField("socketPath", c.SocketPath), "Failed to connect to socketPath")
	}
	c.conn = &conn

	if err := conn.SetDeadline(time.Now().Add(10 * time.Second)); err != nil {
		return errs.WithE(err, "Failed to set deadline")
	}

	if err := WriteBytes(*c.conn, []byte("socket_password ")); err != nil {
		return errs.WithE(err, "Failed to write command")
	}

	if err := c.SocketPassword.Write(*c.conn); err != nil {
		return errs.WithE(err, "Failed to write socket secret")
	}

	if err := WriteBytes(*c.conn, []byte{'\n'}); err != nil {
		return err
	}
	return nil
}

func (c *Client) Close() {
	if c.conn != nil {
		(*c.conn).Close()
		c.conn = nil
	}
}

func (c *Client) SetSecret(secretService *Service) error {
	if c.conn == nil {
		return errs.With("Not connected")
	}

	if err := WriteBytes(*c.conn, []byte("set_secret ")); err != nil {
		return errs.WithE(err, "Failed to write command")
	}

	if err := secretService.Write(*c.conn); err != nil {
		return errs.WithE(err, "Failed to write key")
	}

	if err := WriteBytes(*c.conn, []byte{'\n'}); err != nil {
		return err
	}
	return nil
}

func (c *Client) GetSecret(secretService *Service) error {
	if c.conn == nil {
		return errs.With("Not connected")
	}
	if err := WriteBytes(*c.conn, []byte("get_secret\n")); err != nil {
		return errs.WithE(err, "Failed to write command")
	}

	if err := secretService.FromConnection(*c.conn); err != nil {
		return errs.WithE(err, "Failed to get secret")
	}
	return nil
}
