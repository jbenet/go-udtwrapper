package udt

import (
	"io"
	"unsafe"
)

// #cgo CFLAGS: -Wall
// #cgo LDFLAGS: libudt.a -lstdc++
//
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
func (fd *udtFD) udtIOError() error {
	ec := C.udt_getlasterror_code()
	switch ec {
	case C.UDT_SUCCESS: // success :)
	case C.UDT_ECONNFAIL, C.UDT_ECONNLOST: // connection closed
		// TODO: maybe return some sort of error? this is weird
	case C.UDT_EASYNCRCV, C.UDT_EASYNCSND: // no data to read (async)
	case C.UDT_ETIMEOUT: // timeout that we triggered
	case C.UDT_EINVSOCK:
		// This one actually means that the socket was closed
		return io.EOF
	default: // unexpected error, bail
		return lastError()
	}

	return nil
}

func (fd *udtFD) Read(buf []byte) (int, error) {
	n := int(C.udt_recv(fd.sock, slice2cbuf(buf), C.int(len(buf)), 0))
	if C.int(n) == C.ERROR {
		// got problems?
		return 0, fd.udtIOError()
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
		return 0, fd.udtIOError()
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
