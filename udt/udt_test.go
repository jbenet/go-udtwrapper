package udt

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io"
	"net"
	"sync"
	"testing"
)

func TestStressOps(t *testing.T) {
	addr := getTestAddr()
	l, err := Listen("udt", addr)
	if err != nil {
		t.Fatal(err)
	}

	srcbuf := make([]byte, 50000)
	rand.Read(srcbuf)

	numcons := 200
	numloops := 5000

	var wg sync.WaitGroup
	for i := 0; i < numcons; i++ {
		wg.Add(1)
		go func(nc int) {
			defer wg.Done()
			con, err := Dial("udt", addr)
			if err != nil {
				t.Fatal(err)
			}
			defer con.Close()

			for i := 0; i < numloops; i++ {
				n, err := con.Write(srcbuf[i : i+1024])
				if err != nil {
					t.Fatal(err)
				}
				if n != 1024 {
					t.Fatal("wrote wrong amount")
				}
			}

			fmt.Printf("%d/%d done sending\n", nc, numcons)
		}(i)
	}

	var rwg sync.WaitGroup
	for i := 0; i < numcons; i++ {
		c, err := l.Accept()
		if err != nil {
			t.Fatal(err)
		}

		rwg.Add(1)
		go func(nc int, c net.Conn) {
			defer rwg.Done()
			defer c.Close()
			buf := make([]byte, 1024)
			for i := 0; i < numloops; i++ {
				_, err := io.ReadFull(c, buf)
				if err != nil {
					t.Fatal(err)
				}

				if !bytes.Equal(buf, srcbuf[i:i+1024]) {
					t.Fatal("read wrong data")
				}
			}

			fmt.Printf("%d/%d done receiving\n", nc, numcons)
		}(i, c)
	}

	wg.Wait()
	rwg.Wait()
}
