package memguarded

import (
	"crypto/tls"
	"crypto/x509"
	"log"
	"net"
	"time"

	"github.com/n0rad/go-erlog/data"
	"github.com/n0rad/go-erlog/errs"
	"github.com/n0rad/go-erlog/logs"
)

type Client struct {
	SocketPath     string
	CertPassphrase *Service
	CertKey        string
	CertPem        string

	conn net.Conn
}

func (c *Client) Connect() error {
	cert, err := loadX509KeyPair(c.CertPem, c.CertKey, c.CertPassphrase)
	if err != nil {
		return errs.WithE(err, "Failed to load key pair")
	}

	config := tls.Config{Certificates: []tls.Certificate{cert}, InsecureSkipVerify: true}

	conn, err := tls.Dial("unix", c.SocketPath, &config)
	if err != nil {
		return errs.WithEF(err, data.WithField("socketPath", c.SocketPath), "Failed to connect to socketPath")
	}
	c.conn = net.Conn(conn)

	state := conn.ConnectionState()
	for _, v := range state.PeerCertificates {
		key, err := x509.MarshalPKIXPublicKey(v.PublicKey)
		if err != nil {
			return errs.WithE(err, "failed to marshal public key")
		}
		logs.WithF(data.WithField("key", key)).Debug("Server public key")
	}
	log.Println("client: handshake: ", state.HandshakeComplete)
	log.Println("client: mutual: ", state.NegotiatedProtocolIsMutual)

	if err := conn.SetDeadline(time.Now().Add(10 * time.Second)); err != nil {
		return errs.WithE(err, "Failed to set deadline")
	}

	return nil
}

func (c *Client) Close() {
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
}

func (c *Client) SetSecret(secretService *Service) error {
	if c.conn == nil {
		return errs.With("Not connected")
	}

	if err := WriteBytes(c.conn, []byte("set_secret ")); err != nil {
		return errs.WithE(err, "Failed to write command")
	}

	if err := secretService.Write(c.conn); err != nil {
		return errs.WithE(err, "Failed to write key")
	}

	if err := WriteBytes(c.conn, []byte{'\n'}); err != nil {
		return err
	}
	return nil
}

func (c *Client) GetSecret(secretService *Service) error {
	if c.conn == nil {
		return errs.With("Not connected")
	}
	if err := WriteBytes(c.conn, []byte("get_secret\n")); err != nil {
		return errs.WithE(err, "Failed to write command")
	}

	if err := secretService.FromReaderUntilNewLine(c.conn); err != nil {
		return errs.WithE(err, "Failed to get secret")
	}
	return nil
}
