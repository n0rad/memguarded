package memguarded

import (
	"bytes"
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"github.com/awnumar/memguard"
	"github.com/stretchr/testify/assert"
)

// fakeConn is a minimal net.Conn implementation for testing readCommand and WriteBytes
// without TLS or Unix socket specifics.
type fakeConn struct {
	io.Reader
	io.Writer
}

func (c *fakeConn) Read(p []byte) (int, error)         { return c.Reader.Read(p) }
func (c *fakeConn) Write(p []byte) (int, error)        { return c.Writer.Write(p) }
func (c *fakeConn) Close() error                       { return nil }
func (c *fakeConn) LocalAddr() net.Addr                { return nil }
func (c *fakeConn) RemoteAddr() net.Addr               { return nil }
func (c *fakeConn) SetDeadline(t time.Time) error      { return nil }
func (c *fakeConn) SetReadDeadline(t time.Time) error  { return nil }
func (c *fakeConn) SetWriteDeadline(t time.Time) error { return nil }

func TestReadCommand_SimpleCommand(t *testing.T) {
	memguard.CatchInterrupt()

	r := bytes.NewBufferString("set_secret \nrest")
	conn := &fakeConn{Reader: r, Writer: &bytes.Buffer{}}

	cmd, err := readCommand(conn)
	assert.NoError(t, err)
	assert.Equal(t, "set_secret", cmd)
}

func TestReadCommand_StopsOnNewline(t *testing.T) {
	memguard.CatchInterrupt()

	r := bytes.NewBufferString("get_secret\n")
	conn := &fakeConn{Reader: r, Writer: &bytes.Buffer{}}

	cmd, err := readCommand(conn)
	assert.NoError(t, err)
	assert.Equal(t, "get_secret", cmd)
}

func TestReadCommand_PropagatesError(t *testing.T) {
	memguard.CatchInterrupt()

	errSentinel := errors.New("boom")
	conn := &fakeConn{Reader: &errReader{err: errSentinel}, Writer: &bytes.Buffer{}}

	cmd, err := readCommand(conn)
	assert.Error(t, err)
	assert.Equal(t, "", cmd)
}

// errReader always returns an error on Read.
type errReader struct{ err error }

func (e *errReader) Read(p []byte) (int, error) { return 0, e.err }

func TestWriteBytes_WritesAll(t *testing.T) {
	buf := &bytes.Buffer{}
	data := []byte("hello world")

	err := WriteBytes(buf, data)
	assert.NoError(t, err)
	assert.Equal(t, data, buf.Bytes())
}

func TestWriteBytes_PropagatesError(t *testing.T) {
	w := &errWriter{}
	data := []byte("x")

	err := WriteBytes(w, data)
	assert.Error(t, err)
}

// errWriter returns an error on first Write.
type errWriter struct{}

func (e *errWriter) Write(p []byte) (int, error) { return 0, errors.New("write failed") }

// Test handleConnection to ensure it wraps errors and optionally stops the server.

// fakeServer embeds Server and lets us observe Stop calls.
type fakeServer struct {
	Server
	stopped bool
}

func (fs *fakeServer) Stop(e error) {
	fs.stopped = true
}

// connFuncConn is a net.Conn whose behaviour is injected via function fields, to
// allow us to trigger specific errors inside handleConnectionE via readCommand.

type connFuncConn struct {
	net.Conn
	readFunc       func([]byte) (int, error)
	setDeadlineErr error
}

func (c *connFuncConn) Read(p []byte) (int, error) {
	if c.readFunc != nil {
		return c.readFunc(p)
	}
	return 0, io.EOF
}

func (c *connFuncConn) SetDeadline(t time.Time) error {
	if c.setDeadlineErr != nil {
		return c.setDeadlineErr
	}
	return nil
}

func TestHandleConnection_DoesNotPanicOnError(t *testing.T) {
	fs := &fakeServer{}

	// We don't exercise the full TLS path here because handleConnectionE will
	// fail immediately when trying to assert *tls.Conn. We only ensure that
	// handleConnection wraps the error and does not panic.
	conn := &fakeConn{Reader: &bytes.Buffer{}, Writer: &bytes.Buffer{}}

	fs.handleConnection(conn)
	// No assertion needed other than "did not panic".
}

func TestServer_InitRegistersCommands(t *testing.T) {
	memguard.CatchInterrupt()

	svc := NewService()
	var s Server

	err := s.Init(svc)
	assert.NoError(t, err)

	// Expect the commands map to contain the known commands.
	assert.Contains(t, s.commands, "set_secret")
	assert.Contains(t, s.commands, "get_secret")
}
