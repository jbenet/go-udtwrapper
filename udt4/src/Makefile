C++ = g++

ifndef os
   os = LINUX
endif

ifndef arch
   arch = IA32
endif

CCFLAGS = -fPIC -Wall -Wextra -D$(os) -finline-functions -O3 -fno-strict-aliasing -fvisibility=hidden

ifeq ($(arch), IA32)
   CCFLAGS += -DIA32
endif

ifeq ($(arch), POWERPC)
   CCFLAGS += -mcpu=powerpc
endif

ifeq ($(arch), SPARC)
   CCFLAGS += -DSPARC
endif

ifeq ($(arch), IA64)
   CCFLAGS += -DIA64
endif

ifeq ($(arch), AMD64)
   CCFLAGS += -DAMD64
endif

OBJS = api.o buffer.o cache.o ccc.o channel.o common.o core.o epoll.o list.o md5.o packet.o queue.o window.o
DIR = $(shell pwd)

all: libudt.a ../../libudt.a

%.o: %.cpp %.h udt.h
	$(C++) $(CCFLAGS) $< -c

libudt.a: $(OBJS)
	ar -rcs $@ $^

../../libudt.a:
	cp libudt.a ../../

clean:
	rm -f *.o *.a ../../libudt.a

install:
	export LD_LIBRARY_PATH=$(DIR):$$LD_LIBRARY_PATH
