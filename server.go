package memguarded

import (
	"github.com/n0rad/go-erlog/data"
	"github.com/n0rad/go-erlog/errs"
	"github.com/n0rad/go-erlog/logs"
	"golang.org/x/sys/unix"
	"io"
	"net"
	"os"
	"os/user"
	"strconv"
	"syscall"
	"time"
)

type Server struct {
	Timeout                      time.Duration
	SocketPath                   string
	AnyClientErrorCloseTheServer bool

	userUid  uint32
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

		go s.handleConnection(conn)
	}
}

func (s *Server) Stop(e error) {
	if s.listener != nil {
		_ = s.listener.Close()
	}
	s.stop <- struct{}{}
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

	if s.AnyClientErrorCloseTheServer {
		s.Stop(err)
	}
}

func (s *Server) handleConnectionE(conn net.Conn) error {
	defer func() {
		if err := conn.Close(); err != nil {
			logs.WithE(err).Warn("Socket Connection closed with error")
		}
	}()

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
			if err != io.EOF {
				return errs.WithE(err, "Failed to read command on socket")
			}
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

func getConnectionCredentials(c net.Conn) (*unix.Ucred, error) {
	uc, ok := c.(*net.UnixConn)
	if !ok {
		return nil, errs.With("Not supported socket type")
	}

	raw, err := uc.SyscallConn()
	if err != nil {
		return nil, errs.WithE(err, "Failed to open raw connection")
	}

	var cred *unix.Ucred
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
