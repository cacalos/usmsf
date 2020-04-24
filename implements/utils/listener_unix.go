// +build !windows

package utils

import (
	"fmt"
	"net"
	"runtime"

	"github.com/valyala/fasthttp/reuseport"
)

// GetListener 전달된 주소의 Listener를 반환한다.
func GetListener(network, addr string, port int) (net.Listener, error) {
	lnAddr := fmt.Sprintf("%s:%d", addr, port)
	if runtime.NumCPU() > 1 {
		return reuseport.Listen(network, lnAddr)
	}

	return net.Listen(network, lnAddr)
}
