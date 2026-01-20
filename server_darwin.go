package memguarded

import (
	"net"

	"golang.org/x/sys/unix"
)

func getConnectionCredentials(c net.Conn) (*unix.Ucred, error) {
	// darwin does not support SO_PEERCRED
	return nil, nil
}
