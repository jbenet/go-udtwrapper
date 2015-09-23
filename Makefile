go-deps:
	cd udt4/src && make libudt.a
	cp udt4/src/libudt.a udt/
	cd udt && go get ./...
