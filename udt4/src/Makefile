C++ = g++

ifndef os
   os = $(shell uname)
endif

ifndef arch
   arch = $(shell uname -m)
endif

CCFLAGS = -fPIC -Wall -Wextra -D$(os) -finline-functions -O3 -fno-strict-aliasing -fvisibility=hidden

ifeq ($(arch), ia32)
   CCFLAGS += -DIA32
endif

ifeq ($(arch), POWERPC)
   CCFLAGS += -mcpu=powerpc
endif

ifeq ($(arch), sparc64)
   CCFLAGS += -DSPARC
endif

ifeq ($(arch), ia64)
   CCFLAGS += -DIA64
endif

ifeq ($(arch), x86_64)
   CCFLAGS += -DAMD64
endif

ifeq ($(arch), amd64)
   CCFLAGS += -DAMD64
endif

ifeq ($(arch), ARM)
   CCFLAGS += -DARM
endif

OBJS = api.o buffer.o cache.o ccc.o channel.o common.o core.o epoll.o list.o md5.o packet.o queue.o window.o udt_wrap.o
DIR = $(shell pwd)

all: libudt.so libudt.a udt

%.o: %.cpp %.h udt.h
	$(C++) $(CCFLAGS) $< -c

libudt.so: $(OBJS)
ifneq ($(os), Darwin)
	$(C++) -pthread -shared -o $@ $^
else
	$(C++) -dynamiclib -o libudt.dylib -lstdc++ -lpthread -lm $^ -undefined suppress -flat_namespace
endif

libudt.a: $(OBJS)
	ar -rcs $@ $^

udt:
	cp udt.h udt

clean:
	rm -f *.o *.so *.dylib *.a *.cxx *_gc.c udt

install:
	export LD_LIBRARY_PATH=$(DIR):$$LD_LIBRARY_PATH
