package memguarded

import (
	"crypto/tls"
	"net"

	"github.com/n0rad/go-erlog/errs"
	"golang.org/x/sys/unix"
)

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
