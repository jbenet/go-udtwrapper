package udt

import (
	"net"
	"testing"
	"time"
)

func TestReadTimeout(t *testing.T) {
	addr := getTestAddr()
	buf := make([]byte, 100)
	go func() {
		l, err := Listen("udt", addr)
		if err != nil {
			t.Fatal(err)
		}
		defer l.Close()
		c1, _ := l.Accept()
		c1.Read(buf)
	}()

	c0, err := Dial("udt", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer c0.Close()
	// timeout 1 second
	c0.SetReadDeadline(time.Now().Add(time.Second))
	_, err = c0.Read(buf)
	if err == nil {
		t.Fatal("should not be return succeed")
	}
	if opError, ok := err.(*net.OpError); !ok || !opError.Timeout() {
		t.Fatalf("should be a read timeout error:%s", err)
	}
}
