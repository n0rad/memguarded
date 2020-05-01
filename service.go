package memguarded

import (
	"fmt"
	"github.com/awnumar/memguard"
	"github.com/n0rad/go-erlog/errs"
	"github.com/n0rad/go-erlog/logs"
	"golang.org/x/crypto/ssh/terminal"
	"io"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

type Service struct {
	password   *memguard.Enclave
	notify     map[chan struct{}]struct{}
	notifyLock sync.RWMutex
	stop       chan struct{}
}

func (s *Service) Init() {
	s.notify = make(map[chan struct{}]struct{})
	s.stop = make(chan struct{})
}

func (s *Service) Stop(e error) {
	close(s.stop)
}

func (s *Service) Start() error {
	term := make(chan os.Signal, 1)
	signal.Notify(term, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	select {
	case <-term:
	case <-s.stop:
	}

	logs.Debug("Purge memguard")
	memguard.Purge()
	return nil
}

func (s *Service) FromBytes(password *[]byte) error {
	defer memguard.WipeBytes(*password)
	s.setAndNotify(memguard.NewBufferFromBytes(*password))
	return nil
}

func (s *Service) FromConnection(conn net.Conn) error {
	buffer, err := memguard.NewBufferFromReaderUntil(conn, '\n')
	if err != nil && err != io.EOF {
		return errs.WithE(err, "Failed to read password from connection")
	}
	s.setAndNotify(buffer)
	return nil
}

func (s *Service) AskPassword(confirmation bool) error {
	if !terminal.IsTerminal(int(os.Stdout.Fd())) {
		return errs.With("Cannot ask password, not in a terminal")
	}
	return s.FromStdin(confirmation)
}

func (s *Service) FromStdin(confirmation bool) error {
	var password, passwordConfirm []byte
	defer memguard.WipeBytes(password)
	defer memguard.WipeBytes(passwordConfirm)

	for {
		var err error

		print("Password: ")
		password, err = terminal.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return errs.WithE(err, "Cannot read password")
		}

		print("\n")
		if !confirmation {
			s.setAndNotify(memguard.NewBufferFromBytes(password))
			return nil
		}

		print("Confirm: ")
		passwordConfirm, err = terminal.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return errs.WithE(err, "Cannot read password")
		}
		print("\n")

		if string(password) == string(passwordConfirm) && string(password) != "" {
			s.setAndNotify(memguard.NewBufferFromBytes(password))
			return nil
		} else {
			fmt.Println("\nEmpty password or do not match...\n")
		}
	}
}

func (s *Service) Unwatch(c chan struct{}) {
	s.notifyLock.Lock()
	defer s.notifyLock.Unlock()

	delete(s.notify, c)
}

func (s *Service) Watch() chan struct{} {
	s.notifyLock.Lock()
	defer s.notifyLock.Unlock()

	c := make(chan struct{})
	s.notify[c] = struct{}{}
	return c
}

func (s Service) Write(writer io.Writer) error {
	var total, written int
	var err error

	if s.password == nil {
		return errs.With("Password is not set")
	}

	lockedBuffer, err := s.password.Open()
	if err != nil {
		return errs.WithE(err, "Failed to open password enclave")
	}
	defer lockedBuffer.Destroy()

	bytes := lockedBuffer.Bytes()

	for total = 0; total < len(bytes); total += written {
		written, err = writer.Write(bytes[total:])
		if err != nil {
			return err
		}
	}
	return nil
}

func (s Service) IsSet() bool {
	if s.password != nil {
		return true
	}
	return false
}

func (s Service) Get() (*memguard.LockedBuffer, error) {
	if !s.IsSet() {
		return nil, errs.With("No password set")
	}
	return s.password.Open()
}

/////

func (s *Service) setAndNotify(buffer *memguard.LockedBuffer) {
	s.notifyLock.RLock()
	defer s.notifyLock.RUnlock()

	logs.Debug("Password set")
	s.password = buffer.Seal()
	for e := range s.notify {
		e <- struct{}{}
	}
}
