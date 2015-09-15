package udt

import (
	"errors"
	"fmt"
	"net"
	"runtime"
	"syscall"
	"time"
	"unsafe"

	ctxgrp "github.com/jbenet/go-ctxgroup"
)

// #cgo CFLAGS: -Wall
// #cgo LDFLAGS: libudt.a -lstdc++ -lm
//
// #include "udt_c.h"
// #include <errno.h>
// #include <arpa/inet.h>
// #include <string.h>
import "C"

// returned when calling functions on a fd that's closing.
var errClosing = errors.New("file descriptor closing")

var (
	// UDP_RCVBUF_SIZE is the default UDP_RCVBUF size.
	UDP_RCVBUF_SIZE = uint32(20971520) // 20MB

	// UDT_SNDTIMEO is the udt_send() timeout in milliseconds
	// note this doesnt change the interface, we use it as a poor polling
	UDT_SNDTIMEO_MS = C.int(UDT_ASYNC_TIMEOUT)

	// UDT_RCVTIMEO is the udt_recv() timeout in milliseconds
	// note this doesnt change the interface, we use it as a poor polling
	UDT_RCVTIMEO_MS = C.int(UDT_ASYNC_TIMEOUT)

	// UDT_ASYNC_TIMEOUT (in ms)
	UDT_ASYNC_TIMEOUT = 40
)

func init() {
	// adjust the rcvbuf to our max.
	max, err := maxRcvBufSize()
	if err == nil {
		max = uint32(float32(max) * 0.5) // 0.5 of max.
		if max < UDP_RCVBUF_SIZE {
			UDP_RCVBUF_SIZE = max
		}
	}
}

type signal struct{}
type semaphore chan struct{}

// udtFD (wraps udt.socket)
type udtFD struct {
	csema semaphore
	rsema semaphore
	wsema semaphore
	proc  ctxgrp.ContextGroup

	refcnt int32
	bound  bool

	// immutable until Close
	sock        C.UDTSOCKET
	localClose  bool
	isConnected bool
	net         string
	laddr       *UDTAddr
	raddr       *UDTAddr
}

func newFD(sock C.UDTSOCKET, laddr, raddr *UDTAddr, net string) (*udtFD, error) {
	lac := laddr.copy()
	rac := raddr.copy()
	fd := &udtFD{sock: sock, laddr: lac, raddr: rac, net: net}

	// sync managing.
	fd.csema = make(semaphore, 1)
	fd.rsema = make(semaphore, 1)
	fd.wsema = make(semaphore, 1)
	fd.csema <- signal{}
	fd.rsema <- signal{}
	fd.wsema <- signal{}
	fd.proc = ctxgrp.WithTeardown(fd.teardown)
	fd.proc.AddChildFunc(fd.connectionCheck)
	runtime.SetFinalizer(fd, (*udtFD).Close)
	return fd, nil
}

func (fd *udtFD) connectionCheck(ctxgrp.ContextGroup) {

	for {
		select {
		case <-fd.proc.Closing():
			<-time.After(500 * time.Millisecond) // wait for flushing to happen (TODO)
			<-fd.csema                           // take it forever.
			go closeSocket(fd.sock)
			return

		// check for connection death
		case <-time.After(time.Duration(UDT_ASYNC_TIMEOUT) * time.Millisecond):
			if getSocketStatus(fd.sock).inTeardown() {
				<-time.After(500 * time.Millisecond) // wait for flushing to happen (TODO)
				<-fd.csema                           // take it forever.
				go fd.Close()
				return
			}
		}
	}
}

// lastErrorOp returns the last error as a net.OpError.
func (fd *udtFD) lastErrorOp(op string) *net.OpError {
	return &net.OpError{
		Op:   op,
		Net:  fd.net,
		Addr: fd.laddr,
		Err:  lastError(),
	}
}

func (fd *udtFD) name() string {
	var ls, rs string
	if fd.laddr != nil {
		ls = fd.laddr.String()
	}
	if fd.raddr != nil {
		rs = fd.raddr.String()
	}
	return fd.net + ":" + ls + "->" + rs
}

func (fd *udtFD) setDefaultOpts() error {

	// reduce maximum size
	if C.udt_setsockopt(fd.sock, 0, C.UDP_RCVBUF, unsafe.Pointer(&UDP_RCVBUF_SIZE), C.sizeof_int) != 0 {
		return fmt.Errorf("failed to set UDP_RCVBUF: %d, %s", UDP_RCVBUF_SIZE, lastError())
	}

	// set UDT_REUSEADDR
	trueint := C.int(1)
	if C.udt_setsockopt(fd.sock, 0, C.UDT_REUSEADDR, unsafe.Pointer(&trueint), C.sizeof_int) != 0 {
		return fmt.Errorf("failed to set UDT_REUSEADDR: %s", lastError())
	}

	// unset UDT_LINGER
	// falseint := C.int(0)
	// if C.udt_setsockopt(fd.sock, 0, C.UDT_LINGER, unsafe.Pointer(&falseint), C.sizeof_int) != 0 {
	// 	return fmt.Errorf("failed to set UDT_LINGER: %s", lastError())
	// }

	return nil
}

func (fd *udtFD) setAsyncOpts() error {

	if C.udt_setsockopt(fd.sock, 0, C.UDT_RCVTIMEO, unsafe.Pointer(&UDT_RCVTIMEO_MS), C.sizeof_int) != 0 {
		return fmt.Errorf("failed to set UDT_RCVTIMEO: %s", lastError())
	}

	if C.udt_setsockopt(fd.sock, 0, C.UDT_SNDTIMEO, unsafe.Pointer(&UDT_SNDTIMEO_MS), C.sizeof_int) != 0 {
		return fmt.Errorf("failed to set UDT_SNDTIMEO: %s", lastError())
	}

	// full async is off

	// options
	// falseint := C.int(0)

	// // set sending to be async
	// if C.udt_setsockopt(fd.sock, 0, C.UDT_SNDSYN, unsafe.Pointer(&falseint), C.sizeof_int) != 0 {
	// 	return fmt.Errorf("failed to set UDT_SNDSYN: %s", lastError())
	// }

	// // set receiving to be async
	// if C.udt_setsockopt(fd.sock, 0, C.UDT_RCVSYN, unsafe.Pointer(&falseint), C.sizeof_int) != 0 {
	// 	return fmt.Errorf("failed to set UDT_RCVSYN: %s", lastError())
	// }

	return nil
}

func (fd *udtFD) bind() error {
	_, sa, salen, err := fd.laddr.socketArgs()
	if err != nil {
		return err
	}

	// cast sockaddr
	csa := (*C.struct_sockaddr)(unsafe.Pointer(sa))
	if C.udt_bind(fd.sock, csa, C.int(salen)) != 0 {
		return fd.lastErrorOp("bind")
	}

	return nil
}

func (fd *udtFD) listen(backlog int) error {

	if C.udt_listen(fd.sock, C.int(backlog)) == C.ERROR {
		return fd.lastErrorOp("listen")
	}
	return nil
}

func (fd *udtFD) accept() (*udtFD, error) {
	if err := fd.incref(); err != nil {
		return nil, err
	}
	defer fd.decref()

	var sa syscall.RawSockaddrAny
	var salen C.int

	sock2 := C.udt_accept(fd.sock, (*C.struct_sockaddr)(unsafe.Pointer(&sa)), &salen)
	if sock2 == C.INVALID_SOCK {
		err := fd.lastErrorOp("accept")
		return nil, err
	}

	raddr, err := addrWithSockaddr(&sa)
	if err != nil {
		closeSocket(sock2)
		return nil, fmt.Errorf("error converting address: %s", err)
	}

	remotefd, err := newFD(sock2, fd.laddr, raddr, fd.net)
	if err != nil {
		closeSocket(sock2)
		return nil, err
	}

	if err = remotefd.setAsyncOpts(); err != nil {
		remotefd.Close()
		return nil, err
	}

	return remotefd, nil
}

func (fd *udtFD) connect(raddr *UDTAddr) error {
	if err := fd.incref(); err != nil {
		return err
	}
	defer fd.decref()

	_, sa, salen, err := raddr.socketArgs()
	if err != nil {
		return err
	}
	csa := (*C.struct_sockaddr)(unsafe.Pointer(sa))

	if C.udt_connect(fd.sock, csa, C.int(salen)) == C.ERROR {
		err := lastError()
		return fmt.Errorf("error connecting: %s", err)
	}

	fd.raddr = raddr
	return fd.setAsyncOpts()
}

func (fd *udtFD) Close() error {
	return fd.proc.Close()
}

func (fd *udtFD) teardown() error {
	closeSocket(fd.sock)
	fd.sock = -1
	runtime.SetFinalizer(fd, nil)
	return nil
}

// net.Conn functions

func (fd *udtFD) LocalAddr() net.Addr {
	return fd.laddr
}

func (fd *udtFD) RemoteAddr() net.Addr {
	return fd.raddr
}

func (fd *udtFD) SetDeadline(t time.Time) error {
	panic("not yet implemented")
}

func (fd *udtFD) SetReadDeadline(t time.Time) error {
	panic("not yet implemented")
}

func (fd *udtFD) SetWriteDeadline(t time.Time) error {
	panic("not yet implemented")
}

// lastError returns the last error as a Go string.
func lastError() error {
	return errors.New(C.GoString(C.udt_getlasterror_desc()))
}

func socket(addrfamily int) (sock C.UDTSOCKET, reterr error) {

	sock = C.udt_socket(C.int(addrfamily), C.SOCK_STREAM, 0)
	if sock == C.INVALID_SOCK {
		return C.INVALID_SOCK, fmt.Errorf("invalid socket: %s", lastError())
	}

	return sock, nil
}

func closeSocket(sock C.UDTSOCKET) error {
	if C.udt_close(sock) == C.ERROR {
		return lastError()
	}
	return nil
}

// dialFD sets up a udtFD
func dialFD(laddr, raddr *UDTAddr) (*udtFD, error) {

	if raddr == nil {
		return nil, &net.OpError{Op: "dial", Net: "udt", Addr: raddr, Err: errors.New("invalid remote address")}
	}

	if laddr != nil && laddr.AF() != raddr.AF() {
		return nil, &net.OpError{Op: "dial", Net: "udt", Addr: raddr, Err: errors.New("differing remote address network")}
	}

	sock, err := socket(raddr.AF())
	if err != nil {
		return nil, err
	}

	fd, err := newFD(sock, laddr, raddr, "udt")
	if err != nil {
		closeSocket(sock)
		return nil, err
	}

	if err := fd.setDefaultOpts(); err != nil {
		fd.Close()
		return nil, err
	}

	if laddr != nil {
		if err := fd.bind(); err != nil {
			fd.Close()
			return nil, err
		}
	}

	if err := fd.connect(raddr); err != nil {
		fd.Close()
		return nil, err
	}

	return fd, nil
}

// listenFD sets up a udtFD
func listenFD(laddr *UDTAddr) (*udtFD, error) {

	if laddr == nil {
		return nil, &net.OpError{Op: "dial", Net: "udt", Err: errors.New("invalid address")}
	}

	sock, err := socket(laddr.AF())
	if err != nil {
		return nil, err
	}

	fd, err := newFD(sock, laddr, nil, "udt")
	if err != nil {
		closeSocket(sock)
		return nil, err
	}

	if err := fd.setDefaultOpts(); err != nil {
		fd.Close()
		return nil, err
	}

	if err := fd.bind(); err != nil {
		fd.Close()
		return nil, err
	}

	// TODO: use maxListenerBacklog like golang.org/net/
	if err := fd.listen(syscall.SOMAXCONN); err != nil {
		fd.Close()
		return nil, err
	}

	return fd, nil
}
