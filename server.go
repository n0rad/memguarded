package memguarded

import (
	"fmt"
	"github.com/n0rad/go-erlog/data"
	"github.com/n0rad/go-erlog/errs"
	"github.com/n0rad/go-erlog/logs"
	"io"
	"net"
	"os"
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
		if _, err := conn.Write([]byte{'\n'}); err != nil {
			return err
		}
		return err
	}

	return nil
}

func (s *Server) Start() error {
	s.cleanupSocket()
	s.stop = make(chan struct{}, 1)
	defer close(s.stop)

	listener, err := net.Listen("unix", s.SocketPath)
	if err != nil {
		return errs.WithEF(err, data.WithField("path", s.SocketPath), "Failed to listen on socket")
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
