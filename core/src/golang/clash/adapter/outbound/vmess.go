package outbound

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/Dreamacro/clash/component/dialer"
	"github.com/Dreamacro/clash/component/resolver"
	C "github.com/Dreamacro/clash/constant"
	"github.com/Dreamacro/clash/transport/gun"
	"github.com/Dreamacro/clash/transport/vmess"

	"golang.org/x/net/http2"
)

type Vmess struct {
	*Base
	client *vmess.Client
	option *VmessOption

	// for gun mux
	gunTLSConfig *tls.Config
	gunConfig    *gun.Config
	transport    *http2.Transport
}

type VmessOption struct {
	BasicOption
	Name           string       `proxy:"name"`
	Server         string       `proxy:"server"`
	Port           int          `proxy:"port"`
	UUID           string       `proxy:"uuid"`
	AlterID        int          `proxy:"alterId"`
	Cipher         string       `proxy:"cipher"`
	UDP            bool         `proxy:"udp,omitempty"`
	Network        string       `proxy:"network,omitempty"`
	TLS            bool         `proxy:"tls,omitempty"`
	SkipCertVerify bool         `proxy:"skip-cert-verify,omitempty"`
	ServerName     string       `proxy:"servername,omitempty"`
	HTTPOpts       HTTPOptions  `proxy:"http-opts,omitempty"`
	GrpcOpts       GrpcOptions  `proxy:"grpc-opts,omitempty"`
	WSOpts         WSOptions    `proxy:"ws-opts,omitempty"`
}

type HTTPOptions struct {
	Method  string              `proxy:"method,omitempty"`
	Path    []string            `proxy:"path,omitempty"`
	Headers map[string][]string `proxy:"headers,omitempty"`
}

type HTTP2Options struct {
	Host []string `proxy:"host,omitempty"`
	Path string   `proxy:"path,omitempty"`
}

type GrpcOptions struct {
	GrpcServiceName string `proxy:"grpc-service-name,omitempty"`
}

type WSOptions struct {
	Path                string            `proxy:"path,omitempty"`
	Headers             map[string]string `proxy:"headers,omitempty"`
}

// StreamConn implements C.ProxyAdapter
func (v *Vmess) StreamConn(c net.Conn, metadata *C.Metadata) (net.Conn, error) {
	var err error
	switch v.option.Network {
	case "ws":
		host, port, _ := net.SplitHostPort(v.addr)
		wsOpts := &vmess.WebsocketConfig{
			Host:                host,
			Port:                port,
			Path:                v.option.WSOpts.Path,
		}

		if len(v.option.WSOpts.Headers) != 0 {
			header := http.Header{}
			for key, value := range v.option.WSOpts.Headers {
				header.Add(key, value)
			}
			wsOpts.Headers = header
		}

		if v.option.TLS {
			wsOpts.TLS = true
			wsOpts.TLSConfig = &tls.Config{
				ServerName:         host,
				InsecureSkipVerify: v.option.SkipCertVerify,
				NextProtos:         []string{"http/1.1"},
			}
			if v.option.ServerName != "" {
				wsOpts.TLSConfig.ServerName = v.option.ServerName
			} else if host := wsOpts.Headers.Get("Host"); host != "" {
				wsOpts.TLSConfig.ServerName = host
			}
		}
		c, err = vmess.StreamWebsocketConn(c, wsOpts)
	case "http":
		// readability first, so just copy default TLS logic
		if v.option.TLS {
			host, _, _ := net.SplitHostPort(v.addr)
			tlsOpts := &vmess.TLSConfig{
				Host:           host,
				SkipCertVerify: v.option.SkipCertVerify,
			}

			if v.option.ServerName != "" {
				tlsOpts.Host = v.option.ServerName
			}

			c, err = vmess.StreamTLSConn(c, tlsOpts)
			if err != nil {
				return nil, err
			}
		}

		host, _, _ := net.SplitHostPort(v.addr)
		httpOpts := &vmess.HTTPConfig{
			Host:    host,
			Method:  v.option.HTTPOpts.Method,
			Path:    v.option.HTTPOpts.Path,
			Headers: v.option.HTTPOpts.Headers,
		}

		c = vmess.StreamHTTPConn(c, httpOpts)
	case "grpc":
		c, err = gun.StreamGunWithConn(c, v.gunTLSConfig, v.gunConfig)
	default:
		// handle TLS
		if v.option.TLS {
			host, _, _ := net.SplitHostPort(v.addr)
			tlsOpts := &vmess.TLSConfig{
				Host:           host,
				SkipCertVerify: v.option.SkipCertVerify,
			}

			if v.option.ServerName != "" {
				tlsOpts.Host = v.option.ServerName
			}

			c, err = vmess.StreamTLSConn(c, tlsOpts)
		}
	}

	if err != nil {
		return nil, err
	}

	return v.client.StreamConn(c, parseVmessAddr(metadata))
}

// DialContext implements C.ProxyAdapter
func (v *Vmess) DialContext(ctx context.Context, metadata *C.Metadata) (_ C.Conn, err error) {
	// gun transport
	if v.transport != nil {
		c, err := gun.StreamGunWithTransport(v.transport, v.gunConfig)
		if err != nil {
			return nil, err
		}
		defer safeConnClose(c, err)

		c, err = v.client.StreamConn(c, parseVmessAddr(metadata))
		if err != nil {
			return nil, err
		}

		return NewConn(c, v), nil
	}

	c, err := dialer.DialContext(ctx, "tcp", v.addr)
	if err != nil {
		return nil, err
	}
	defer safeConnClose(c, err)

	c, err = v.StreamConn(c, metadata)
	return NewConn(c, v), err
}

// ListenPacketContext implements C.ProxyAdapter
func (v *Vmess) ListenPacketContext(ctx context.Context, metadata *C.Metadata) (_ C.PacketConn, err error) {
	// vmess use stream-oriented udp with a special address, so we needs a net.UDPAddr
	if !metadata.Resolved() {
		ip, err := resolver.ResolveIP(metadata.Host)
		if err != nil {
			return nil, errors.New("can't resolve ip")
		}
		metadata.DstIP = ip
	}

	var c net.Conn
	// gun transport
	if v.transport != nil {
		c, err = gun.StreamGunWithTransport(v.transport, v.gunConfig)
		if err != nil {
			return nil, err
		}
		defer safeConnClose(c, err)

		c, err = v.client.StreamConn(c, parseVmessAddr(metadata))
	} else {
		c, err = dialer.DialContext(ctx, "tcp", v.addr)
		if err != nil {
			return nil, err
		}
		defer safeConnClose(c, err)

		c, err = v.StreamConn(c, metadata)
	}

	if err != nil {
		return nil, err
	}

	return newPacketConn(&vmessPacketConn{Conn: c, rAddr: metadata.UDPAddr()}, v), nil
}

func NewVmess(option VmessOption) (*Vmess, error) {
	security := strings.ToLower(option.Cipher)
	client, err := vmess.NewClient(vmess.Config{
		UUID:     option.UUID,
		AlterID:  uint16(option.AlterID),
		Security: security,
		HostName: option.Server,
		Port:     strconv.Itoa(option.Port),
		IsAead:   option.AlterID == 0,
	})
	if err != nil {
		return nil, err
	}

	switch option.Network {
	case "grpc":
		if !option.TLS {
			return nil, nil
		}
	}

	v := &Vmess{
		Base: &Base{
			name:  option.Name,
			addr:  net.JoinHostPort(option.Server, strconv.Itoa(option.Port)),
			tp:    C.Vmess,
			udp:   option.UDP,
		},
		client: client,
		option: &option,
	}

	switch option.Network {
	case "grpc":
		dialFn := func(network, addr string) (net.Conn, error) {
			c, err := dialer.DialContext(context.Background(), "tcp", v.addr)
			if err != nil {
				return nil, err
			}
			return c, nil
		}

		gunConfig := &gun.Config{
			ServiceName: v.option.GrpcOpts.GrpcServiceName,
			Host:        v.option.ServerName,
		}
		tlsConfig := &tls.Config{
			InsecureSkipVerify: v.option.SkipCertVerify,
			ServerName:         v.option.ServerName,
		}

		if v.option.ServerName == "" {
			host, _, _ := net.SplitHostPort(v.addr)
			tlsConfig.ServerName = host
			gunConfig.Host = host
		}

		v.gunTLSConfig = tlsConfig
		v.gunConfig = gunConfig
		v.transport = gun.NewHTTP2Client(dialFn, tlsConfig)
	}

	return v, nil
}

func parseVmessAddr(metadata *C.Metadata) *vmess.DstAddr {
	var addrType byte
	var addr []byte
	switch metadata.AddrType {
	case C.AtypIPv4:
		addrType = byte(vmess.AtypIPv4)
		addr = make([]byte, net.IPv4len)
		copy(addr[:], metadata.DstIP.To4())
	case C.AtypIPv6:
		addrType = byte(vmess.AtypIPv6)
		addr = make([]byte, net.IPv6len)
		copy(addr[:], metadata.DstIP.To16())
	case C.AtypDomainName:
		addrType = byte(vmess.AtypDomainName)
		addr = make([]byte, len(metadata.Host)+1)
		addr[0] = byte(len(metadata.Host))
		copy(addr[1:], []byte(metadata.Host))
	}

	port, _ := strconv.ParseUint(metadata.DstPort, 10, 16)
	return &vmess.DstAddr{
		UDP:      metadata.NetWork == C.UDP,
		AddrType: addrType,
		Addr:     addr,
		Port:     uint(port),
	}
}

type vmessPacketConn struct {
	net.Conn
	rAddr net.Addr
}

func (uc *vmessPacketConn) WriteTo(b []byte, addr net.Addr) (int, error) {
	return uc.Conn.Write(b)
}

func (uc *vmessPacketConn) ReadFrom(b []byte) (int, net.Addr, error) {
	n, err := uc.Conn.Read(b)
	return n, uc.rAddr, err
}
