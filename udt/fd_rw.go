package udt

import (
	"io"
	"net"
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
	// switch C.udt_getlasterror_code() {
	// case C.UDT_SUCCESS: // success :)
	// case C.UDT_ECONNFAIL, C.UDT_ECONNLOST: // connection closed
	// case C.UDT_EASYNCRCV, C.UDT_EASYNCSND: // no data to read (async)
	// case C.UDT_ETIMEOUT: // timeout that we triggered
	// default: // unexpected error, bail
	//  return lastError()
	// }

	// timeout and/or closing.

	// TODO remove this and turn async off. This timeout is here because I'm seeing
	// unexpected blocking (violating the timeout). Its not clear how the UDT async
	// stuff and Goroutines mesh... this worked.
	// UPDATE: async disabled for now.
	select {
	// case <-time.After(time.Duration(UDT_ASYNC_TIMEOUT) * time.Millisecond):
	case <-fd.proc.Closing():
		return io.EOF // seems to have been a graceful close
	default:
	}
	return nil
}

func (fd *udtFD) incref() error {
	select {
	case <-fd.proc.Closing():
		return errClosing
	case <-fd.csema:
		fd.proc.Children().Add(1)
		fd.csema <- signal{}
		return nil
	}
}

func (fd *udtFD) decref() {
	fd.proc.Children().Done()
}

func (fd *udtFD) readLock() error {
	// first acquire control sema to add ourselves as a child
	if err := fd.incref(); err != nil {
		return err
	}

	// second acquire read sema (one reader at a time)
	select {
	case <-fd.proc.Closing():
		fd.decref() // didnt work out. undo
		return errClosing
	case <-fd.rsema:
		return nil
	}
}

func (fd *udtFD) readUnlock() {
	fd.decref()
	fd.rsema <- signal{}
}

func (fd *udtFD) writeLock() error {
	select {
	case <-fd.proc.Closing():
		return errClosing
	case <-fd.csema:
		<-fd.wsema
		fd.proc.Children().Add(1)
		fd.csema <- signal{}
		return nil
	}
}

func (fd *udtFD) writeUnlock() {
	fd.proc.Children().Done()
	fd.wsema <- signal{}
}

func (fd *udtFD) Read(buf []byte) (readcnt int, err error) {
	if err = fd.readLock(); err != nil {
		return 0, err
	}
	defer fd.readUnlock()

	readcnt = 0
	for {
		n := int(C.udt_recv(fd.sock, slice2cbuf(buf[readcnt:]), C.int(len(buf)-readcnt), 0))
		if C.int(n) == C.ERROR {
			// got problems?
			if err = fd.udtIOError(); err != nil {
				break
			}
			// nope, everything's fine. read again.
			continue
		}

		if n > 0 {
			readcnt += n
		}
		if err != nil { // bad things happened
			break
		}
		if n == 0 {
			err = io.EOF
		}
		break // return the data we have.
	}
	if err != nil && err != io.EOF {
		err = &net.OpError{Op: "read", Net: fd.net, Addr: fd.laddr, Err: err}
	}
	return readcnt, err
}

func (fd *udtFD) Write(buf []byte) (writecnt int, err error) {
	if err = fd.writeLock(); err != nil {
		return 0, err
	}
	defer fd.writeUnlock()

	writecnt = 0
	for {
		n := int(C.udt_send(fd.sock, slice2cbuf(buf[writecnt:]), C.int(len(buf)-writecnt), 0))

		if C.int(n) == C.ERROR {
			// UDT Error?
			if err = fd.udtIOError(); err != nil {
				break
			}
			// everything's fine, proceed
		}

		// update our running count
		if n > 0 {
			writecnt += n
		}

		if writecnt == len(buf) { // done!
			break
		}
		if err != nil { // bad things happened
			break
		}
		if n == 0 { // early eof?
			err = io.ErrUnexpectedEOF
			break
		}
	}
	if err != nil {
		err = &net.OpError{Op: "write", Net: fd.net, Addr: fd.raddr, Err: err}
	}
	return writecnt, err
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
