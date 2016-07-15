package udt

import (
	"io"
	"net"
	"os"
	"syscall"
	"unsafe"
)

// #include "udt_c.h"
// #include <errno.h>
// #include <arpa/inet.h>
// #include <string.h>
import "C"

func slice2cbuf(buf []byte) *C.char {
	return (*C.char)(unsafe.Pointer(&buf[0]))
}

// udtIOError interprets the udt_getlasterror_code and returns an
// error if IO systems should stop.
func (fd *udtFD) udtIOError(op string) error {
	ec := C.udt_getlasterror_code()
	switch ec {
	case C.UDT_SUCCESS: // success :)
		fallthrough
	case C.UDT_ECONNFAIL, C.UDT_ECONNLOST: // connection closed
		// TODO: maybe return some sort of error? this is weird
		fallthrough
	case C.UDT_EASYNCRCV, C.UDT_EASYNCSND: // no data to read (async)
		fallthrough
	case C.UDT_EINVSOCK:
		// This one actually means that the socket was closed
		return io.EOF
	case C.UDT_ETIMEOUT: // timeout that we triggered
		return &net.OpError{Op: op, Net: "udt", Source: fd.laddr, Addr: fd.raddr, Err: os.NewSyscallError(op, syscall.ETIMEDOUT)}
	default: // unexpected error, bail
		return lastError()
	}

	return nil
}

func (fd *udtFD) Read(buf []byte) (int, error) {
	n := int(C.udt_recv(fd.sock, slice2cbuf(buf), C.int(len(buf)), 0))
	if C.int(n) == C.ERROR {
		// got problems?
		return 0, fd.udtIOError("read")
	}
	return n, nil
}

func (fd *udtFD) Write(buf []byte) (writecnt int, err error) {
	for len(buf) > writecnt {
		n, err := fd.write(buf[writecnt:])
		if err != nil {
			return writecnt, err
		}

		writecnt += n
	}
	return writecnt, nil
}

func (fd *udtFD) write(buf []byte) (int, error) {
	n := int(C.udt_send(fd.sock, slice2cbuf(buf), C.int(len(buf)), 0))
	if C.int(n) == C.ERROR {
		// UDT Error?
		return 0, fd.udtIOError("write")
	}

	return n, nil
}

type socketStatus C.enum_UDTSTATUS

func getSocketStatus(sock C.UDTSOCKET) socketStatus {
	return socketStatus(C.udt_getsockstate(sock))
}

func (s socketStatus) inSetup() bool {
	switch C.enum_UDTSTATUS(s) {
	case C.INIT, C.OPENED, C.LISTENING, C.CONNECTING:
		return true
	}
	return false
}

func (s socketStatus) inTeardown() bool {
	switch C.enum_UDTSTATUS(s) {
	case C.BROKEN, C.CLOSED, C.NONEXIST: // c.CLOSING
		return true
	}
	return false
}

func (s socketStatus) inConnected(sock C.UDTSOCKET) bool {
	return C.enum_UDTSTATUS(s) == C.CONNECTED
}
