package proxy

import (
	"net"
	"strconv"
	"sync"

	"github.com/Dreamacro/clash/listener/http"
)

var (
	httpListener      *http.Listener

	// lock for recreate function
	httpMux   sync.Mutex
)

type Ports struct {
	Port       int `json:"port"`
}

// GetPorts return the ports of proxy servers
func GetPorts() *Ports {
	ports := &Ports{}

	if httpListener != nil {
		_, portStr, _ := net.SplitHostPort(httpListener.Address())
		port, _ := strconv.Atoi(portStr)
		ports.Port = port
	}

	return ports
}

func portIsZero(addr string) bool {
	_, port, err := net.SplitHostPort(addr)
	if port == "0" || port == "" || err != nil {
		return true
	}
	return false
}
