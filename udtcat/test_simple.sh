#!/bin/sh

set -e # exit on error
# set -v # verbose

# the program name
netcat=udtcat

log() {
  echo "--> $1"
}

test_send() {
  file=$1_
  count=$2
  addr=localhost:8765

  # generate random data
  log "generating $count bytes of random data"
  dd if=/dev/random of=${file}expected bs=$count count=1

  # dialer sends
  log "sending from dialer"
  ./$netcat -v $addr 2>&1 <${file}expected | sed "s/^/  dialer1: /" &
  dialer=$!
  ./$netcat -v -l $addr 2>&1 >${file}actual1 | sed "s/^/listener1: /"
  listener=$!

  diff ${file}expected ${file}actual1
  if test $? != 0; then
    log "sending from dialer failed. compare with:\n"
    log "diff ${file}expected ${file}actual1"
    kill -9 $dialer
    kill -9 $listener
    exit 1
  fi

  # listener sends
  log "sending from listener"
  ./$netcat -v -l $addr 2>&1 <${file}expected | sed "s/^/listener2: /" &
  dialer=$!
  ./$netcat -v $addr 2>&1 >${file}actual2 | sed "s/^/  dialer2: /"
  listener=$!
  diff ${file}expected ${file}actual2
  if test $? != 0; then
    log "sending from listener failed. compare with:\n"
    log "diff ${file}expected ${file}actual2"
    kill -9 $dialer
    kill -9 $listener
    exit 1
  fi

  echo rm ${file}{expected,actual1,actual2}
  rm ${file}{expected,actual1,actual2}
  return 0
}

mkdir -p ".trash"
test_send ".trash/1KB" 1024
test_send ".trash/1MB" 1048576
test_send ".trash/1GB" 1073741824
