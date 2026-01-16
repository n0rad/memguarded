package memguarded

import (
	"bytes"
	"io"
	"net"
	"testing"
	"time"

	"github.com/awnumar/memguard"
	"github.com/stretchr/testify/assert"
)

func TestService_FromBytesAndGet(t *testing.T) {
	memguard.CatchInterrupt()

	svc := NewService()
	b := []byte("super-secret")

	assert.NoError(t, svc.FromBytes(&b))
	assert.True(t, svc.IsSet())

	locked, err := svc.Get()
	assert.NoError(t, err)
	t.Cleanup(locked.Destroy)

	assert.Equal(t, []byte("super-secret"), locked.Bytes())
}

func TestService_Write(t *testing.T) {
	memguard.CatchInterrupt()

	svc := NewService()
	b := []byte("hello-world")
	assert.NoError(t, svc.FromBytes(&b))

	var buf bytes.Buffer
	assert.NoError(t, svc.Write(&buf))
	assert.Equal(t, "hello-world", buf.String())
}

func TestService_Write_NoSecret(t *testing.T) {
	memguard.CatchInterrupt()

	svc := NewService()
	var buf bytes.Buffer

	err := svc.Write(&buf)
	assert.Error(t, err)
}

// mockConn is a minimal net.Conn implementation backed by an io.Reader for tests
// We only implement Read because FromReaderUntilNewLine only calls Read.
type mockConn struct {
	io.Reader
}

func (m *mockConn) Read(p []byte) (int, error) { return m.Reader.Read(p) }

// Satisfy net.Conn interface with no-op implementations
func (m *mockConn) Write(b []byte) (n int, err error) { return len(b), nil }
func (m *mockConn) Close() error                      { return nil }
func (m *mockConn) LocalAddr() net.Addr               { return nil }
func (m *mockConn) RemoteAddr() net.Addr              { return nil }
func (m *mockConn) SetDeadline(t time.Time) error     { return nil }
func (m *mockConn) SetReadDeadline(t time.Time) error { return nil }
func (m *mockConn) SetWriteDeadline(t time.Time) error {
	return nil
}

func TestService_FromReaderUntilNewLine(t *testing.T) {
	memguard.CatchInterrupt()

	svc := NewService()
	data := []byte("line-content\nignored")
	conn := &mockConn{Reader: bytes.NewReader(data)}

	assert.NoError(t, svc.FromReaderUntilNewLine(conn))

	locked, err := svc.Get()
	assert.NoError(t, err)
	t.Cleanup(locked.Destroy)

	assert.Equal(t, []byte("line-content"), locked.Bytes())
}

func TestService_Reader_ReadAll(t *testing.T) {
	memguard.CatchInterrupt()

	svc := NewService()
	b := []byte("read-all-secret")
	assert.NoError(t, svc.FromBytes(&b))

	r := svc.Reader()
	assert.NotNil(t, r)

	all, err := io.ReadAll(r)
	assert.NoError(t, err)
	assert.Equal(t, []byte("read-all-secret"), all)
}

func TestService_Reader_NilWhenNotSet(t *testing.T) {
	memguard.CatchInterrupt()

	var svc Service
	assert.False(t, svc.IsSet())
	assert.Nil(t, svc.Reader())
}

func TestSecretReader_ErrorOnNilEnclave(t *testing.T) {
	memguard.CatchInterrupt()

	r := &secretReader{}
	buf := make([]byte, 10)
	_, err := r.Read(buf)
	assert.Error(t, err)
}

func TestService_WatchAndUnwatch(t *testing.T) {
	memguard.CatchInterrupt()

	svc := NewService()

	ch := svc.Watch()

	b := []byte("trigger")
	go func() { _ = svc.FromBytes(&b) }()

	select {
	case <-ch:
		// ok
	case <-time.After(time.Second):
		t.Fatalf("did not receive notification")
	}

	svc.Unwatch(ch)
}
