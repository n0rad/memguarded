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
	"sync"
	"syscall"
)

type Service struct {
	secret     *memguard.Enclave
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
	memguard.CatchInterrupt()
	<-s.stop
	return nil
}

func (s *Service) FromBytes(secret *[]byte) error {
	defer memguard.WipeBytes(*secret)
	s.setAndNotify(memguard.NewBufferFromBytes(*secret))
	return nil
}

func (s *Service) FromReaderUntilNewLine(conn net.Conn) error {
	buffer, err := memguard.NewBufferFromReaderUntil(conn, '\n')
	if err != nil && err != io.EOF {
		return errs.WithE(err, "Failed to read secret from connection")
	}
	s.setAndNotify(buffer)
	return nil
}

func (s *Service) AskSecret(confirmation bool, name string) error {
	if !terminal.IsTerminal(int(os.Stdout.Fd())) {
		return errs.With("Cannot ask secret, not in a terminal")
	}
	return s.FromStdin(confirmation, name)
}

func (s *Service) FromStdin(confirmation bool, name string) error {
	var secret, secretConfirm []byte
	defer memguard.WipeBytes(secret)
	defer memguard.WipeBytes(secretConfirm)

	for {
		var err error

		print(name + ": ")
		secret, err = terminal.ReadPassword(syscall.Stdin)
		if err != nil {
			return errs.WithE(err, "Cannot read secret")
		}

		print("\n")
		if !confirmation {
			s.setAndNotify(memguard.NewBufferFromBytes(secret))
			return nil
		}

		print("Confirm: ")
		secretConfirm, err = terminal.ReadPassword(syscall.Stdin)
		if err != nil {
			return errs.WithE(err, "Cannot read secret")
		}
		print("\n")

		if string(secret) == string(secretConfirm) && string(secret) != "" {
			s.setAndNotify(memguard.NewBufferFromBytes(secret))
			return nil
		} else {
			fmt.Println("\nEmpty secret or do not match...")
			fmt.Println()
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

	if s.secret == nil {
		return errs.With("Secret is not set")
	}

	lockedBuffer, err := s.secret.Open()
	if err != nil {
		return errs.WithE(err, "Failed to open secret enclave")
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
	if s.secret != nil {
		return true
	}
	return false
}

func (s Service) Get() (*memguard.LockedBuffer, error) {
	if !s.IsSet() {
		return nil, errs.With("No secret set")
	}
	return s.secret.Open()
}

/////

func (s *Service) setAndNotify(buffer *memguard.LockedBuffer) {
	s.notifyLock.RLock()
	defer s.notifyLock.RUnlock()

	logs.Debug("Secret set")
	s.secret = buffer.Seal()
	for e := range s.notify {
		e <- struct{}{}
	}
}
