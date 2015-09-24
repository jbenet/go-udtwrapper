package udt

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"runtime"
	"syscall"
	"testing"
)

func TestSocketConstruct(t *testing.T) {
	if _, err := socket(syscall.AF_INET); err != nil {
		t.Fatal(err)
	}
}

func TestSocketClose(t *testing.T) {
	s, err := socket(syscall.AF_INET)
	assert(t, nil == err, err)

	if int(s) <= 0 {
		t.Fatal("socket num invalid")
	}

	if err := closeSocket(s); err != nil {
		t.Fatal(err)
	}

	if err := closeSocket(s); err == nil {
		t.Fatal("closing again did not produce error")
	}
}

func TestUdtFDConstruct(t *testing.T) {
	a, err := ResolveUDTAddr("udt", ":1234")
	assert(t, nil == err, err)
	s, err := socket(a.AF())
	assert(t, nil == err, err)

	if int(s) <= 0 {
		t.Fatal("socket num invalid")
	}

	fd, err := newFD(s, a, nil, "udt")
	if err != nil {
		t.Fatal(err)
	}

	if fd.name() != "udt::1234->" {
		t.Fatal("incorrect name:", fd.name())
	}

	if err := fd.Close(); err != nil {
		t.Fatal(err)
	}

	if int(fd.sock) != -1 {
		t.Fatal("sock should now be -1")
	}
}

/* TODO: this test doesnt make any sense
and feels like its just testing some goprocess stuff that we're not using anymore
func TestUdtFDLocking(t *testing.T) {
	a, err := ResolveUDTAddr("udt", ":1234")
	assert(t, nil == err, err)
	s, err := socket(a.AF())
	assert(t, nil == err, err)
	fd, err := newFD(s, a, nil, "udt")
	assert(t, nil == err, err)
	err = fd.setDefaultOpts()
	assert(t, nil == err, err)

	go func() {
		if err := fd.Close(); err != nil {
			t.Fatal(err)
		}
		done <- signal{}
	}()

	select {
	case <-done:
		t.Fatal("sock should not have happened yet")
	default:
	}

	if int(fd.sock) == -1 {
		t.Fatal("sock should not yet be -1")
	}

	fd.proc.Children().Done()
	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("sock should have happened now")
	}

	if int(fd.sock) != -1 {
		t.Fatal("sock should now be -1")
	}
}
*/

func TestUdtFDListenOnly(t *testing.T) {
	addr := getTestAddr()
	la, err := ResolveUDTAddr("udt", addr)
	assert(t, nil == err, err)
	s, err := socket(la.AF())
	assert(t, nil == err, err)
	fd, err := newFD(s, la, nil, "udt")
	assert(t, nil == err, err)

	if err := fd.listen(10); err == nil {
		t.Fatal("should fail. must bind first")
	}

	if err := fd.bind(); err != nil {
		t.Fatal(err)
	}

	if err := fd.listen(10); err != nil {
		t.Fatal(err)
	}

	if err := fd.Close(); err != nil {
		t.Fatal(err)
	}

	assert(t, fd.sock == -1, "sock should now be -1", fd.sock)
}

func TestUdtFDAcceptAndConnect(t *testing.T) {
	addr := getTestAddr()
	al, err := ResolveUDTAddr("udt", addr)
	assert(t, nil == err, err)
	sl, err := socket(al.AF())
	assert(t, nil == err, err)
	sc, err := socket(al.AF())
	assert(t, nil == err, err)
	fdl, err := newFD(sl, al, nil, "udt")
	assert(t, nil == err, err)
	fdc, err := newFD(sc, nil, nil, "udt")
	assert(t, nil == err, err)
	err = fdl.bind()
	assert(t, nil == err, err)
	err = fdl.listen(10)
	assert(t, nil == err, err)

	cerrs := make(chan error, 10)
	go func() {
		defer close(cerrs)

		err := fdc.connect(al)
		if err != nil {
			cerrs <- err
			return
		}

		if fdc.raddr != al {
			cerrs <- fmt.Errorf("addr should be set (todo change)")
		}

		cerrs <- fdc.Close()

		if err := fdc.connect(al); err == nil {
			cerrs <- fmt.Errorf("should not be able to connect after closing")
		}

		assert(t, fdc.sock == -1, "sock should now be -1", fdc.sock)
	}()

	connl, err := fdl.accept()
	if err != nil {
		t.Fatal(err)
	}

	if connl.sock <= 0 {
		t.Fatal("sock <= 0", connl.sock)
	}

	if err := fdl.Close(); err != nil {
		t.Fatal(err)
	}

	assert(t, fdl.listen(10) != nil, "should not be able to listen after closing")
	assert(t, fdl.sock == -1, "sock should now be -1", fdl.sock)

	// drain connector errs
	for err := range cerrs {
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestUdtFDAcceptAndDialFD(t *testing.T) {
	addr := getTestAddr()
	al, err := ResolveUDTAddr("udt", addr)
	assert(t, nil == err, err)
	sl, err := socket(al.AF())
	assert(t, nil == err, err)
	fdl, err := newFD(sl, al, nil, "udt")
	assert(t, nil == err, err)
	err = fdl.bind()
	assert(t, nil == err, err)
	err = fdl.listen(10)
	assert(t, nil == err, err)

	cerrs := make(chan error, 10)
	go func() {
		defer close(cerrs)

		fdc, err := dialFD(nil, al)
		if err != nil {
			fmt.Printf("failed to dial %s", err)
			cerrs <- err
			return
		}

		if fdc.raddr != al {
			cerrs <- fmt.Errorf("addr should be set (todo change)")
		}

		cerrs <- fdc.Close()

		if err := fdc.connect(al); err == nil {
			cerrs <- fmt.Errorf("should not be able to connect after closing")
		}

		assert(t, fdc.sock == -1, "sock should now be -1", fdc.sock)
	}()

	connl, err := fdl.accept()
	if err != nil {
		t.Fatal(err)
	}

	if connl.sock <= 0 {
		t.Fatal("sock <= 0", connl.sock)
	}

	if err := fdl.Close(); err != nil {
		t.Fatal(err)
	}

	assert(t, fdl.listen(10) != nil, "should not be able to listen after closing")
	assert(t, fdl.sock == -1, "sock should now be -1", fdl.sock)

	// drain connector errs
	for err := range cerrs {
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestUdtDialFDAndListenFD(t *testing.T) {
	addr := getTestAddr()
	al, err := ResolveUDTAddr("udt", addr)
	assert(t, nil == err, err)

	cerrs := make(chan error, 10)
	go func() {
		defer close(cerrs)
		fdc, err := dialFD(nil, al)
		if err != nil {
			cerrs <- err
			return
		}

		if fdc.raddr != al {
			cerrs <- fmt.Errorf("addr should be set (todo change)")
		}

		cerrs <- fdc.Close()

		if err := fdc.connect(al); err == nil {
			cerrs <- fmt.Errorf("should not be able to connect after closing")
		}

		assert(t, fdc.sock == -1, "sock should now be -1", fdc.sock)
	}()

	fdl, err := listenFD(al)
	if err != nil {
		t.Fatal(err)
	}

	connl, err := fdl.accept()
	if err != nil {
		t.Fatal(err)
	}

	if connl.sock <= 0 {
		t.Fatal("sock <= 0", connl.sock)
	}

	err = connl.Close()
	assert(t, nil == err, err)

	if err := fdl.Close(); err != nil {
		t.Fatal(err)
	}

	assert(t, fdl.listen(10) != nil, "should not be able to listen after closing")
	assert(t, fdl.sock == -1, "sock should now be -1", fdl.sock)

	// drain connector errs
	for err := range cerrs {
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestUdtReadWrite(t *testing.T) {
	addr := getTestAddr()
	al, err := ResolveUDTAddr("udt", addr)
	assert(t, nil == err, err)

	cerrs := make(chan error, 10)
	go func() {
		defer close(cerrs)
		fdc, err := dialFD(nil, al)
		assert(t, nil == err, err)

		_, err = io.Copy(fdc, fdc)
		if err != nil {
			cerrs <- err
		}
	}()

	fdl, err := listenFD(al)
	assert(t, nil == err, err)

	connl, err := fdl.accept()
	assert(t, nil == err, err)

	testSendToEcho(t, connl)

	err = connl.Close()
	assert(t, nil == err, err)

	err = fdl.Close()
	assert(t, nil == err, err)

	assert(t, fdl.listen(10) != nil, "should not be able to listen after closing")
	assert(t, fdl.sock == -1, "sock should now be -1", fdl.sock)

	// drain connector errs
	for err := range cerrs {
		if err != nil {
			t.Fatal(err)
		}
	}
}

func assert(t *testing.T, cond bool, vals ...interface{}) {
	if !cond {
		_, file, line, _ := runtime.Caller(1)
		prefix := fmt.Sprintf("%s:%d: ", file, line)

		toprint := append([]interface{}{prefix}, vals...)
		t.Fatal(toprint)
	}
}

func testSendToEcho(t *testing.T, conn net.Conn) {

	// the meat of the test is here vvv

	buflen := 1024 * 12
	buf := make([]byte, buflen)

	for i := 0; i < 128; i++ {
		for j := range buf {
			buf[j] = byte('a' + (i % 26))
		}

		nn, err := conn.Write(buf)
		if err != nil {
			t.Fatal(err)
		}
		if nn != buflen {
			t.Fatal("wrote wrong number of bytes", nn, buflen)
		}

		buf2 := make([]byte, buflen)
		n, err := io.ReadFull(conn, buf2)
		if err == io.EOF {
			break
		} else if err != nil {
			t.Fatal(err)
		}
		if n != buflen {
			t.Fatal("read wrong number of bytes somehow")
		}

		if !bytes.Equal(buf, buf2) {
			t.Fatal("bufs differ:\n\n%s\n\n%s", string(buf), string(buf2))
		}
	}

	// the meat of the test is here ^^^
}
