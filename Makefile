go-deps:
	cd udt4/src && make libudt.a
	cp udt4/src/libudt.a udt/
	cd udt && CGO_LDFLAGS=-L. go get ./...
