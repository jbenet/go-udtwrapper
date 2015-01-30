package udt

import (
	"io/ioutil"
	"os"
	"strconv"
)

func maxRcvBufSize() (uint32, error) {
	fi, err := os.Open("/proc/sys/net/core/rmem_max")
	if err != nil {
		return 0, err
	}
	defer fi.Close()

	val, err := ioutil.ReadAll(fi)
	if err != nil {
		return 0, err
	}

	i, err := strconv.Atoi(string(val))
	if err != nil {
		return 0, err
	}

	return uint32(i), nil
}
