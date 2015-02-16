// +build darwin dragonfly freebsd netbsd openbsd

package udt

import "syscall"

func maxRcvBufSize() (uint32, error) {
  return syscall.SysctlUint32("kern.ipc.maxsockbuf")
}
