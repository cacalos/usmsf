// +build windows

package utils

import (
	"fmt"
	"net"
)

// GetListener 전달된 주소의 Listener를 반환한다.
func GetListener(network, addr string, port int) (net.Listener, error) {
	lnAddr := fmt.Sprintf("%s:%d", addr, port)
	return net.Listen(network, lnAddr)
}
