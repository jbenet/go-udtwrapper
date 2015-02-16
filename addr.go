package udt

import (
	"fmt"
	"net"
	"syscall"

	sockaddr "github.com/jbenet/go-sockaddr"
	sockaddrnet "github.com/jbenet/go-sockaddr/net"
)

type UDTAddr struct {
	addr *net.UDPAddr
}

func (a *UDTAddr) Network() string { return "udt" }

func (a *UDTAddr) String() string {
	if a == nil || a.addr == nil {
		return "<nil>"
	}
	return a.addr.String()
}

func (a *UDTAddr) toAddr() net.Addr {
	if a == nil || a.addr == nil {
		return nil
	}
	return a.addr
}

func (a *UDTAddr) copy() *UDTAddr {
	if a == nil || a.addr == nil {
		return nil
	}

	var udp net.UDPAddr
	udp = *a.addr
	return &UDTAddr{addr: &udp}
}

// AF returns UDTAddr's AF (Address Family)
func (a *UDTAddr) AF() int {
	af := sockaddrnet.NetAddrAF(a.addr)
	if af == syscall.AF_UNSPEC {
		af = syscall.AF_INET
	}
	return af
}

// IPPROTO returns UDTAddr's IPPROTO (IPPROTO_UDP)
func (a *UDTAddr) IPPROTO() int {
	return sockaddrnet.NetAddrAF(a.addr)
}

func udt2udp(n string) (string, error) {
	switch n {
	case "udt":
		return "udp", nil
	case "udt4":
		return "udp4", nil
	case "udt6":
		return "udp6", nil
	default:
		return "", net.UnknownNetworkError(n)
	}
}

func ResolveUDTAddr(n, addr string) (*UDTAddr, error) {
	udpnet, err := udt2udp(n)
	if err != nil {
		return nil, err
	}
	udp, err := net.ResolveUDPAddr(udpnet, addr)
	if err != nil {
		return nil, err
	}
	return &UDTAddr{addr: udp}, nil
}

func WrapUDPAddr(ua *net.UDPAddr) *UDTAddr {
	return &UDTAddr{addr: ua}
}

// sockArgs returns (AF, *RawSockaddrAny, error)
func (a *UDTAddr) socketArgs() (int, *syscall.RawSockaddrAny, sockaddr.Socklen, error) {
	af := a.AF()
	sa := sockaddrnet.NetAddrToSockaddr(a.addr)
	if sa == nil {
		return 0, nil, 0, fmt.Errorf("could not convert UDPAddr to syscall.Sockaddr")
	}

	rsa, salen, err := sockaddr.SockaddrToAny(sa)
	if err != nil {
		return 0, nil, 0, fmt.Errorf("could not convert syscall.Sockaddr to syscall.RawSockaddrAny")
	}

	return af, rsa, salen, nil
}

func addrWithSockaddr(rsa *syscall.RawSockaddrAny) (*UDTAddr, error) {
	sa, err := sockaddr.AnyToSockaddr(rsa)
	if err != nil {
		return nil, err
	}

	udpaddr := sockaddrnet.SockaddrToUDPAddr(sa)
	if udpaddr == nil {
		return nil, fmt.Errorf("could not convert syscall.Sockaddr to UDPAddr")
	}
	return &UDTAddr{addr: udpaddr}, nil
}
