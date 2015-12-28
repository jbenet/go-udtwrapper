package udt

import (
	"net"
	"testing"
	"time"
)

func TestWriteTimeout(t *testing.T) {
	addr := "127.0.0.1:38926"
	buf := make([]byte, 100)
	go func() {
		l, err := Listen("udt", addr)
		if err != nil {
			t.Fatal(err)
		}
		defer l.Close()
		c2, err := l.Accept()
		c2.Read(buf)
	}()

	c, err := Dial("udt", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	// timeout 1 second
	c.SetReadDeadline(time.Now().Add(time.Second))
	_, err = c.Read(buf)
	if err == nil {
		t.Fatal("should not be return succeed")
	}
	if opError, ok := err.(*net.OpError); !ok || !opError.Timeout() {
		t.Fatal("should be a timeout error")
	}
}
