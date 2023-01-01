package proxy

import (
	"net"
	"strconv"

	"github.com/Dreamacro/clash/listener/http"
)

var (
	httpListener      *http.Listener
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
