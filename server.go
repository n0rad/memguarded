package memguarded

import (
	"fmt"
	"github.com/n0rad/go-erlog/data"
	"github.com/n0rad/go-erlog/errs"
	"github.com/n0rad/go-erlog/logs"
	"golang.org/x/sys/unix"
	"io"
	"log"
	"net"
	"os"
	"os/user"
	"strconv"
	"syscall"
	"time"
)

type Server struct {
	Timeout    time.Duration
	SocketPath string

	commands map[string]func(net.Conn) error
	stop     chan struct{}
	listener net.Listener
}

func (s *Server) Init(passService *Service) error {
	if s.SocketPath == "" {
		return errs.With("socket path must be set")
	}
	s.Timeout = 10 * time.Second
	s.commands = make(map[string]func(conn net.Conn) error)
	s.commands["set_password"] = func(conn net.Conn) error {
		if err := passService.FromConnection(conn); err != nil {
			return err
		}
		return nil
	}
	s.commands["get_password"] = func(conn net.Conn) error {
		err := passService.Write(conn)
		if err != nil {
			return err
		}
		return WriteBytes(conn, []byte{'\n'})
	}

	return nil
}

func (s *Server) Start() error {
	s.cleanupSocket()
	s.stop = make(chan struct{}, 1)

	listener, err := net.Listen("unix", s.SocketPath)
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

		socketStat, err := os.Stat(s.SocketPath)
		if err != nil {
			return errs.WithEF(err, data.WithField("socket", s.SocketPath), "Failed to get socket stats")
		}
		if socketStat.Mode() != os.ModeSocket|0700 {
			return errs.WithF(data.WithField("socket", s.SocketPath).WithField("mode", socketStat.Mode()).WithField("xx", os.FileMode(0700)), "Socket mod changed")
		}

		if err != nil {
			select {
			case <-s.stop:
				return nil
			default:
				logs.WithE(err).Error("Failed to accept socket connection")
			}
			continue
		}
		go s.handleConnection(conn)
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
	defer func() {
		if err := conn.Close(); err != nil {
			logs.WithE(err).Warn("Socket Connection closed with error")
		}
	}()

	creds, err := readCreds(conn)
	if err != nil {
		logs.WithE(err).Warn("Failed to read client credentials")
		return
	}

	// TODO
	uidStr, err := user.Current()
	if err != nil {

		log.Fatal(err)
	}

	uid, err := strconv.Atoi(uidStr.Uid)
	if err != nil {
		log.Fatal(err)
	}


	if creds.Uid != uint32(uid) {
		logs.WithField("uid", creds.Uid).Error("Unauthorized access")
		//client.Write([]byte("Unauthorized access\n"))
		//client.Close()
		return
	}

	for {
		if err := conn.SetDeadline(time.Now().Add(s.Timeout)); err != nil {
			logs.WithEF(err, data.WithField("timeout", s.Timeout)).Warn("Failed to set deadline on socket connection")
		}

		command, err := readCommand(conn)
		if err != nil {
			if err != io.EOF {
				logs.WithE(err).Error("Failed to read command on socket")
			}
			return
		}

		commandFunc, ok := s.commands[command]
		if !ok {
			_, _ = fmt.Fprintf(conn, "Unknown command %s\n", command)
			return
		}

		if err := commandFunc(conn); err != nil {
			logs.WithE(err).Debug("Client connection command failed")
		}
	}
}

func readCommand(conn net.Conn) (string, error) {
	command := ""
	buffer := make([]byte, 1)
	for {
		n, err := conn.Read(buffer)
		if err == io.EOF {
			return command, nil
		}
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

func readCreds(c net.Conn) (*unix.Ucred, error) {

	var cred *unix.Ucred

	// net.Conn is an interface. Expect only *net.UnixConn types
	uc, ok := c.(*net.UnixConn)
	if !ok {
		return nil, fmt.Errorf("unexpected socket type")
	}

	// Fetches raw network connection from UnixConn
	raw, err := uc.SyscallConn()
	if err != nil {
		return nil, fmt.Errorf("error opening raw connection: %s", err)
	}

	// The raw.Control() callback does not return an error directly.
	// In order to capture errors, we wrap already defined variable
	// 'err' within the closure. 'err2' is then the error returned
	// by Control() itself.
	err2 := raw.Control(func(fd uintptr) {
		cred, err = unix.GetsockoptUcred(int(fd),
			unix.SOL_SOCKET,
			unix.SO_PEERCRED)
	})

	if err != nil {
		return nil, fmt.Errorf("GetsockoptUcred() error: %s", err)
	}

	if err2 != nil {
		return nil, fmt.Errorf("Control() error: %s", err2)
	}

	return cred, nil
}
