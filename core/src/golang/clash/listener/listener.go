package proxy

import (
	"fmt"
	"net"
	"strconv"
	"sync"

	"github.com/Dreamacro/clash/adapter/inbound"
	C "github.com/Dreamacro/clash/constant"
	"github.com/Dreamacro/clash/listener/http"
)

var (
	allowLan    = false
	bindAddress = "*"

	httpListener      *http.Listener
	redirListener     *redir.Listener

	// lock for recreate function
	httpMux   sync.Mutex
	tproxyMux sync.Mutex
)

type Ports struct {
	Port       int `json:"port"`
	SocksPort  int `json:"socks-port"`
	RedirPort  int `json:"redir-port"`
	MixedPort  int `json:"mixed-port"`
}

func AllowLan() bool {
	return allowLan
}

func BindAddress() string {
	return bindAddress
}

func SetAllowLan(al bool) {
	allowLan = al
}

func SetBindAddress(host string) {
	bindAddress = host
}

func ReCreateHTTP(port int, tcpIn chan<- C.ConnContext) {
	httpMux.Lock()
	defer httpMux.Unlock()
	
	var err error
	addr := genAddr(bindAddress, port, allowLan)

	if httpListener != nil {
		if httpListener.RawAddress() == addr {
			return
		}
		httpListener.Close()
		httpListener = nil
	}

	if portIsZero(addr) {
		return
	}

	httpListener, err = http.New(addr, tcpIn)
	if err != nil {
		return
	}
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

func genAddr(host string, port int, allowLan bool) string {
	if allowLan {
		if host == "*" {
			return fmt.Sprintf(":%d", port)
		}
		return fmt.Sprintf("%s:%d", host, port)
	}

	return fmt.Sprintf("127.0.0.1:%d", port)
}
