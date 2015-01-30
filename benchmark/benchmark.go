package main

import (
	"flag"
	"fmt"
	udt "github.com/fffw/go-udtwrapper"
	"io"
	"io/ioutil"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var verbose = false

// Usage prints out the usage of this module.
// Assumes flags use go stdlib flag pacakage.
var Usage = func() {
	text := `benchmark - UDT/TCP benchmark tool

Usage:

  server: %s -s<udt address> <tcp address>
  benchmarker:   %s [udt | tcp] <remote address>

Address format is Go's: [host]:port
`

	fmt.Fprintf(os.Stderr, text, os.Args[0], os.Args[0])
	flag.PrintDefaults()
}

type args struct {
	verbose bool
	listen  bool
	bs      int64

	udtAddr string
	tcpAddr string

	dialUdt    bool
	dialTcp    bool
	remoteAddr string
}

func parseArgs() args {
	var a args

	// setup + parse flags
	flag.BoolVar(&a.listen, "server", false, "listen for connections")
	flag.BoolVar(&a.listen, "s", false, "listen for connections (short)")
	flag.BoolVar(&a.dialUdt, "udt", false, "use udt client")
	flag.BoolVar(&a.dialTcp, "tcp", false, "use tcp client")
	flag.BoolVar(&a.verbose, "v", false, "verbose debugging")
	flag.Int64Var(&a.bs, "bs", 65536, "block size to send and receive")
	flag.Usage = Usage
	flag.Parse()
	osArgs := flag.Args()

	if a.listen {
		if len(osArgs) < 2 {
			Usage()
			exit("")
		}
		a.udtAddr = osArgs[0]
		a.tcpAddr = osArgs[1]
	} else {
		if len(osArgs) < 1 {
			Usage()
			exit("")
		}
		a.remoteAddr = osArgs[0]
	}

	return a
}

func main() {
	args := parseArgs()
	verbose = args.verbose

	go func() {
		// wait until we exit.
		sigc := make(chan os.Signal, 1)
		signal.Notify(sigc, syscall.SIGABRT)
		<-sigc
		panic("ABORT! ABORT! ABORT!")
	}()

	var err error
	if args.listen {
		err = Listen(args.udtAddr, args.tcpAddr, args.bs)
	} else {
		var c net.Conn
		if args.dialUdt {
			c, err = Dial("udt", args.remoteAddr)
		} else {
			c, err = Dial("tcp", args.remoteAddr)
		}
		if err != nil {
			exit("%s", err)
		}
		err = benchmark(c, args.bs)
	}

	if err != nil {
		exit("%s", err)
	}
}

func exit(format string, vals ...interface{}) {
	if format != "" {
		fmt.Fprintf(os.Stderr, "benchmark error: "+format+"\n", vals...)
	}
	os.Exit(1)
}

func log(format string, vals ...interface{}) {
	if verbose {
		fmt.Fprintf(os.Stderr, "benchmark log: "+format+"\n", vals...)
	}
}

// Listen listens and accepts one incoming UDT connection on a given port,
// and pipes all incoming data to os.Stdout.
func Listen(udtAddr string, tcpAddr string, bs int64) error {
	udt, err := udt.Listen("udt", udtAddr)
	if err != nil {
		return err
	}
	log("udt listening at %s", udt.Addr())

	tcp, err := net.Listen("tcp", tcpAddr)
	if err != nil {
		return err
	}
	log("tcp listening at %s", tcp.Addr())

	done := make(chan interface{}, 2)
	run := func(l net.Listener) (err error) {
		for {
			var c net.Conn
			c, err = l.Accept()
			if err != nil {
				return
			}
			log("accepted connection from %s", c.RemoteAddr())

			go func() {
				for {
					select {
					case <-done:
						c.Close()
						return
					default:
						n, e := io.CopyN(c, c, bs)
						if e == io.EOF {
							c.Close()
							return
						}
						log("Copied back %d bytes", n)
					}
				}

				return
			}()
		}
	}

	go run(udt)
	go run(tcp)
	// wait until we exit.
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGHUP, syscall.SIGINT,
		syscall.SIGTERM, syscall.SIGQUIT)
	<-sigc
	fmt.Println("Quit now")
	done <- nil
	return nil
}

// Dial connects to a remote address and pipes all os.Stdin to the remote end.
// If localAddr is set, uses it to Dial from.
func Dial(network, remoteAddr string) (c net.Conn, err error) {
	log("%s dialing %s", network, remoteAddr)
	if network == "udt" {
		c, err = udt.Dial("udt", remoteAddr)
	} else {
		c, err = net.Dial("tcp", remoteAddr)
	}
	if err != nil {
		return
	}
	log("connected to %s", c.RemoteAddr())
	return

}

func benchmark(c net.Conn, bs int64) (err error) {
	defer c.Close()
	rand, err := os.Open("/dev/zero")
	if err != nil {
		return
	}
	log("piping random to connection")

	done := make(chan interface{}, 2)
	var sent, recved int64
	startTime := time.Now()

	reportStat := func(prefix string, bytes int64) {
		timeSpent := time.Now().Sub(startTime) / time.Second
		fmt.Printf("%s %d bytes in %d sec, %d Bps\n", prefix, bytes, timeSpent, bytes/int64(timeSpent))
	}
	go func() {
		ticker := time.Tick(time.Second)
		for {
			select {
			case <-ticker:
				reportStat("Sent", sent)
			case <-done:
				return
			default:
				n, _ := io.CopyN(c, rand, bs)
				sent = sent + n
			}
		}
	}()
	go func() {
		ticker := time.Tick(time.Second)
		for {
			select {
			case <-ticker:
				reportStat("Recv", recved)
			case <-done:
				return
			default:
				n, _ := io.CopyN(ioutil.Discard, c, bs)
				recved = recved + n
			}
		}
	}()

	// wait until we exit.
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGHUP, syscall.SIGINT,
		syscall.SIGTERM, syscall.SIGQUIT)
	<-sigc
	fmt.Println("Quit now, final result:")
	done <- nil
	done <- nil
	reportStat("Sent", sent)
	reportStat("Recv", recved)
	c.Close()
	return
}
