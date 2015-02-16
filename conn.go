package udt

import (
// "net"

// sockaddr "github.com/jbenet/go-sockaddr"
// sockaddrnet "github.com/jbenet/go-sockaddr/net"
)

// UDTConn is the implementation of the Conn and PacketConn interfaces
// for UDT network connections.
type UDTConn struct {
	*udtFD
	net string
}

func newUDTConn(fd *udtFD, net string) *UDTConn {
	return &UDTConn{udtFD: fd, net: net}
}

func dialConn(laddr, raddr *UDTAddr) (*UDTConn, error) {
	fd, err := dialFD(laddr, raddr)
	if err != nil {
		return nil, err
	}

	return newUDTConn(fd, fd.laddr.Network()), nil
}
