package memguarded

import (
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"github.com/n0rad/go-erlog/data"
	"github.com/n0rad/go-erlog/errs"
	"github.com/n0rad/go-erlog/logs"
	"golang.org/x/sys/unix"
	"io"
	"io/ioutil"
	"net"
	"os"
	"os/user"
	"reflect"
	"strconv"
	"syscall"
	"time"
	"unsafe"
)

type Server struct {
	Timeout              time.Duration
	SocketPath           string
	StopOnAnyClientError bool
	CertKey              string
	CertPem              string
	CAPem				 string

	userUid  uint32
	commands map[string]func(net.Conn) error
	stop     chan struct{}
	listener net.Listener
}

func (s *Server) Init(secretService *Service) error {
	s.Timeout = 10 * time.Second
	s.commands = make(map[string]func(net.Conn) error)

	s.commands["set_secret"] = func(m net.Conn) error {
		logs.Info("Set secret")
		if err := secretService.FromReaderUntilNewLine(m); err != nil {
			return err
		}
		return nil
	}
	s.commands["get_secret"] = func(m net.Conn) error {
		logs.Info("Get secret")
		err := secretService.Write(m)
		if err != nil {
			return err
		}
		return WriteBytes(m, []byte{'\n'})
	}

	uidStr, err := user.Current()
	if err != nil {
		return errs.WithE(err, "Failed to get current user uid")
	}

	uid, err := strconv.Atoi(uidStr.Uid)
	if err != nil {
		return errs.WithE(err, "Failed to convert current user uid to int")
	}
	s.userUid = uint32(uid)

	return nil
}

func (s *Server) Start() error {
	s.cleanupSocket()
	s.stop = make(chan struct{}, 1)

	cert, err := tls.LoadX509KeyPair(s.CertPem, s.CertKey)
	if err != nil {
		return errs.WithE(err, "Failed to load server key")
	}

	certpool := x509.NewCertPool()
	pem, err := ioutil.ReadFile(s.CAPem)
	if err != nil {
		return errs.WithE(err, "Failed to read client CA certificate authority")
	}
	if !certpool.AppendCertsFromPEM(pem) {
		return errs.WithE(err, "failed to parse client CA certificate authority")
	}

	config := tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    certpool,
		Rand:         rand.Reader,
	}

	listener, err := tls.Listen("unix", s.SocketPath, &config)
	if err != nil {
		return errs.WithEF(err, data.WithField("path", s.SocketPath), "Failed to listen on socket")
	}
	if err := os.Chmod(s.SocketPath, os.ModeSocket|0700); err != nil {
		return errs.WithEF(err, data.WithField("socket", s.SocketPath), "Failed to set socket permissions")
	}

	s.listener = listener
	defer s.cleanupSocket()

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.stop:
				return nil
			default:
				logs.WithE(err).Error("Failed to accept socket connection")
			}
			continue
		}

		socketStat, err := os.Stat(s.SocketPath)
		if err != nil {
			return errs.WithEF(err, data.WithField("socket", s.SocketPath), "Failed to get socket stats")
		}
		if socketStat.Mode() != os.ModeSocket|0700 {
			return errs.WithF(data.WithField("socket", s.SocketPath).WithField("mode", socketStat.Mode()).WithField("xx", os.FileMode(0700)), "Socket mod changed")
		}

		//go s.handleConnection(conn)
		s.handleConnection(conn) // we don't need a high throughput server
	}
}

func (s *Server) Stop(e error) {
	s.stop <- struct{}{}
	if s.listener != nil {
		_ = s.listener.Close()
	}
}

/////////////////////

func (s *Server) cleanupSocket() {
	_, err := os.Stat(s.SocketPath)
	if os.IsNotExist(err) {
		return
	}

	if err := syscall.Unlink(s.SocketPath); err != nil {
		logs.WithEF(err, data.WithField("path", s.SocketPath)).Warn("Failed to unlink socket")
	}
}

func (s *Server) handleConnection(conn net.Conn) {
	err := s.handleConnectionE(conn)
	if err != nil {
		logs.WithE(err).Error("Client connection handle failed")
	}

	if err != nil && s.StopOnAnyClientError {
		s.Stop(err)
	}
}

func (s *Server) handleConnectionE(conn net.Conn) error {
	defer conn.Close()

	tlscon, ok := conn.(*tls.Conn)
	if !ok {
		return errs.With("Connection is not tls")
	}

	err := tlscon.Handshake()
	if err != nil {
		return errs.WithE(err, "TLS handshare failed")
	}

	state := tlscon.ConnectionState()
	for _, v := range state.PeerCertificates {
		key, err := x509.MarshalPKIXPublicKey(v.PublicKey)
		if err != nil {
			return errs.WithE(err, "failed to marshal public key")
		}
		logs.WithF(data.WithField("key", key)).Debug("Client public key")
	}

	creds, err := getConnectionCredentials(conn)
	if err != nil {
		return errs.WithE(err, "Failed to read client credentials")
	}

	if creds.Uid != s.userUid {
		return errs.WithEF(err, data.WithField("uid", creds.Uid), "Unauthorized access")
	}

	for {
		if err := conn.SetDeadline(time.Now().Add(s.Timeout)); err != nil {
			return errs.WithE(err, "Failed to set deadline on socket connection")
		}

		command, err := readCommand(conn)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return errs.WithE(err, "Failed to read command on socket")
		}

		commandFunc, ok := s.commands[command]
		if !ok {
			return errs.WithEF(err, data.WithField("command", command), "Unknown command on socket")
		}

		if err := commandFunc(conn); err != nil {
			return errs.WithE(err, "Client command failed")
		}
	}
}

func readCommand(conn net.Conn) (string, error) {
	command := ""
	buffer := make([]byte, 1)
	for {
		n, err := conn.Read(buffer)
		if err != nil {
			return "", err
		}
		if n == 0 {
			return "", err
		}

		if string(buffer) == " " || string(buffer) == "\n" {
			return command, nil
		}
		command += string(buffer)
	}
}

func WriteBytes(conn io.Writer, bytes []byte) error {
	var total, written int
	var err error
	for total = 0; total < len(bytes); total += written {
		written, err = conn.Write(bytes[total:])
		if err != nil {
			return err
		}
	}
	return nil
}

func unixConnFromTLSConn(conn tls.Conn) (*net.UnixConn, error) {
	value := reflect.ValueOf(conn)
	field := value.FieldByName("conn")

	// Fails with "reflect.Value.UnsafeAddr of unaddressable value" because rs isn't addressable:
	// field = reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem()

	valueCopy := reflect.New(value.Type()).Elem()
	valueCopy.Set(value)
	field = valueCopy.FieldByName("conn")
	field = reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem()

	i := field.Interface()
	uc, ok := i.(*net.UnixConn)
	if !ok {
		return nil, errs.With("Failed to get unix connection from tls connection")
	}
	return uc, nil
}

func getConnectionCredentials(c net.Conn) (*unix.Ucred, error) {
	var cred *unix.Ucred

	tlscon, ok := c.(*tls.Conn)
	if !ok {
		return cred, errs.With("Connection is not tls")
	}
	uc, err := unixConnFromTLSConn(*tlscon)
	if err != nil {
		return nil, err
	}

	raw, err := uc.SyscallConn()
	if err != nil {
		return nil, errs.WithE(err, "Failed to open raw connection")
	}

	err2 := raw.Control(func(fd uintptr) {
		cred, err = unix.GetsockoptUcred(int(fd),
			unix.SOL_SOCKET,
			unix.SO_PEERCRED)
	})

	if err != nil {
		return nil, errs.WithE(err, "Failed to user credentials from socket control")
	}

	if err2 != nil {
		return nil, errs.WithE(err2, "Failed to user credentials from socket control")
	}

	return cred, nil
}
