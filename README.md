# go-udtwrapper

This is a cgo wrapper around the main C++ UDT implementation.

## Usage

- Godoc: https://godoc.org/github.com/jbenet/go-udtwrapper/udt

## Tools:

- [udtcat](udtcat/) - netcat using the udt pkg

## Try:

```sh
(cd udt4/src; make -e os=OSX arch=AMD64) # this will produce ./libudt.a, for other platform, refer udt4/README.txt
(cd udtcat; go build; ./test_simple.sh)
```
