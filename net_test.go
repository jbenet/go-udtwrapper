package udt

import (
	"fmt"
	"io"
	"testing"
)

func TestResolveUDTAddr(t *testing.T) {
	a, err := ResolveUDTAddr("udt", ":1234")
	if err != nil {
		t.Fatal(err)
	}

	if a.Network() != "udt" {
		t.Fatal("addr resolved incorrectly: %s", a.Network())
	}

	if a.String() != ":1234" {
		t.Fatal("addr resolved incorrectly: %s", a)
	}
}

func TestListenOnly(t *testing.T) {
	l, err := Listen("udt", ":2235")
	if err != nil {
		t.Fatal(err)
	}

	if err := l.Close(); err != nil {
		t.Fatal(err)
	}

	_, err = l.Accept()
	assert(t, err != nil, "should not be able to accept after closing")
}

func TestListenAndDial(t *testing.T) {
	l, err := Listen("udt", "127.0.0.1:2335")
	if err != nil {
		t.Fatal(err)
	}

	cerrs := make(chan error, 10)
	go func() {
		defer close(cerrs)
		c1, err := Dial("udt", "127.0.0.1:2335")
		if err != nil {
			cerrs <- err
			return
		}

		if c1.RemoteAddr().String() != l.Addr().String() {
			cerrs <- fmt.Errorf("addrs should be the same")
		}

		cerrs <- c1.Close()
	}()

	c2, err := l.Accept()
	if err != nil {
		t.Fatal(err)
	}

	if c2.LocalAddr().String() != l.Addr().String() {
		t.Fatal("addrs should be the same")
	}

	if err := c2.Close(); err != nil {
		t.Fatal(err)
	}
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}

	_, err = l.Accept()
	assert(t, err != nil, "should not be able to accept after closing")

	// drain connector errs
	for err := range cerrs {
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestConnReadWrite(t *testing.T) {
	al, err := ResolveUDTAddr("udt", "127.0.0.1:2534")
	assert(t, nil == err, err)

	cerrs := make(chan error, 10)
	go func() {
		defer close(cerrs)
		c2, err := Dial("udt", "127.0.0.1:2534")
		assert(t, nil == err, err)

		n, err := io.Copy(c2, c2)
		fmt.Printf("echoed %d bytes\n", n)
		if err != nil {
			cerrs <- err
		}

		err = c2.Close()
		assert(t, err == nil, err)
	}()

	l, err := Listen("udt", al.String())
	assert(t, nil == err, err)

	c1, err := l.Accept()
	assert(t, nil == err, err)

	testSendToEcho(t, c1)

	err = c1.Close()
	assert(t, nil == err, err)

	err = l.Close()
	assert(t, nil == err, err)

	_, err = l.Accept()
	assert(t, err != nil, "should not be able to listen after closing")

	fmt.Printf("closed and waiting\n")
	// drain connector errs
	for err := range cerrs {
		if err != nil {
			t.Fatal(err)
		}
	}
	fmt.Printf("done\n")
}
