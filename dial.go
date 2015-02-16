package udt

import (
	"errors"
	"net"
)

type Dialer struct {
	LocalAddr net.Addr
}

var errMissingAddress = errors.New("missing address")

// DialUDT connects to the remote address raddr on the network net,
// which must be "udt", "udt4", or "udt6".
func (d *Dialer) DialUDT(network string, raddr *UDTAddr) (*UDTConn, error) {
	switch network {
	case "udt", "udt4", "udt6":
	default:
		return nil, &net.OpError{Op: "dial", Net: network, Addr: raddr, Err: net.UnknownNetworkError(network)}
	}
	if raddr == nil {
		return nil, &net.OpError{Op: "dial", Net: network, Addr: nil, Err: errMissingAddress}
	}

	laddr, ok := d.LocalAddr.(*UDTAddr)
	if !ok {
		laddr = nil
	}

	return dialConn(laddr, raddr)
}

// Dial connects to the remote address raddr on the network net,
// which must be "udt", "udt4", or "udt6".
func (d *Dialer) Dial(network, address string) (c net.Conn, err error) {
	raddr, err := ResolveUDTAddr(network, address)
	if err != nil {
		return nil, err
	}
	return d.DialUDT(network, raddr)
}

// Dial connects to the remote address raddr on the network net,
// which must be "udt", "udt4", or "udt6".  If laddr is not nil, it is
// used as the local address for the connection.
func Dial(network, address string) (c net.Conn, err error) {
	return (&Dialer{}).Dial(network, address)
}

// DialUDT connects to the remote address raddr on the network net,
// which must be "udt", "udt4", or "udt6".  If laddr is not nil, it is
// used as the local address for the connection.
func DialUDT(net string, laddr, raddr *UDTAddr) (*UDTConn, error) {
	return (&Dialer{LocalAddr: laddr}).DialUDT(net, raddr)
}
