package outbound

import (
	"context"
	"errors"
	"net"
	"strconv"

	"github.com/Dreamacro/clash/common/structure"
	"github.com/Dreamacro/clash/component/dialer"
	C "github.com/Dreamacro/clash/constant"
	"github.com/Dreamacro/clash/transport/shadowsocks/core"
	obfs "github.com/Dreamacro/clash/transport/simple-obfs"
	"github.com/Dreamacro/clash/transport/socks5"
)

type ShadowSocks struct {
	*Base
	cipher core.Cipher

	// obfs
	obfsMode    string
	obfsOption  *simpleObfsOption
}

type ShadowSocksOption struct {
	BasicOption
	Name       string         `proxy:"name"`
	Server     string         `proxy:"server"`
	Port       int            `proxy:"port"`
	Password   string         `proxy:"password"`
	Cipher     string         `proxy:"cipher"`
	UDP        bool           `proxy:"udp,omitempty"`
	Plugin     string         `proxy:"plugin,omitempty"`
	PluginOpts map[string]any `proxy:"plugin-opts,omitempty"`
}

type simpleObfsOption struct {
	Mode string `obfs:"mode,omitempty"`
	Host string `obfs:"host,omitempty"`
}

type v2rayObfsOption struct {
	Mode           string            `obfs:"mode"`
	Host           string            `obfs:"host,omitempty"`
	Path           string            `obfs:"path,omitempty"`
	TLS            bool              `obfs:"tls,omitempty"`
	Headers        map[string]string `obfs:"headers,omitempty"`
	SkipCertVerify bool              `obfs:"skip-cert-verify,omitempty"`
	Mux            bool              `obfs:"mux,omitempty"`
}

// StreamConn implements C.ProxyAdapter
func (ss *ShadowSocks) StreamConn(c net.Conn, metadata *C.Metadata) (net.Conn, error) {
	switch ss.obfsMode {
	case "tls":
		c = obfs.NewTLSObfs(c, ss.obfsOption.Host)
	case "http":
		_, port, _ := net.SplitHostPort(ss.addr)
		c = obfs.NewHTTPObfs(c, ss.obfsOption.Host, port)
	}
	c = ss.cipher.StreamConn(c)
	_, err := c.Write(serializesSocksAddr(metadata))
	return c, err
}

// DialContext implements C.ProxyAdapter
func (ss *ShadowSocks) DialContext(ctx context.Context, metadata *C.Metadata) (_ C.Conn, err error) {
	c, err := dialer.DialContext(ctx, "tcp", ss.addr)
	if err != nil {
		return nil, err
	}

	defer safeConnClose(c, err)

	c, err = ss.StreamConn(c, metadata)
	return NewConn(c, ss), err
}

// ListenPacketContext implements C.ProxyAdapter
func (ss *ShadowSocks) ListenPacketContext(ctx context.Context, metadata *C.Metadata) (C.PacketConn, error) {
	pc, err := dialer.ListenPacket(ctx, "udp", "")
	if err != nil {
		return nil, err
	}

	addr, err := resolveUDPAddr("udp", ss.addr)
	if err != nil {
		pc.Close()
		return nil, err
	}

	pc = ss.cipher.PacketConn(pc)
	return newPacketConn(&ssPacketConn{PacketConn: pc, rAddr: addr}, ss), nil
}

func NewShadowSocks(option ShadowSocksOption) (*ShadowSocks, error) {
	addr := net.JoinHostPort(option.Server, strconv.Itoa(option.Port))
	cipher := option.Cipher
	password := option.Password
	ciph, err := core.PickCipher(cipher, nil, password)
	if err != nil {
		return nil, err
	}

	var obfsOption *simpleObfsOption
	obfsMode := ""

	decoder := structure.NewDecoder(structure.Option{TagName: "obfs", WeaklyTypedInput: true})
	if option.Plugin == "obfs" {
		opts := simpleObfsOption{Host: "bing.com"}
		if err := decoder.Decode(option.PluginOpts, &opts); err != nil {
			return nil, err
		}

		if opts.Mode != "tls" && opts.Mode != "http" {
			return nil, err
		}
		obfsMode = opts.Mode
		obfsOption = &opts
	}

	return &ShadowSocks{
		Base: &Base{
			name:  option.Name,
			addr:  addr,
			tp:    C.Shadowsocks,
			udp:   option.UDP,
		},
		cipher: ciph,

		obfsMode:    obfsMode,
		obfsOption:  obfsOption,
	}, nil
}

type ssPacketConn struct {
	net.PacketConn
	rAddr net.Addr
}

func (spc *ssPacketConn) WriteTo(b []byte, addr net.Addr) (n int, err error) {
	packet, err := socks5.EncodeUDPPacket(socks5.ParseAddrToSocksAddr(addr), b)
	if err != nil {
		return
	}
	return spc.PacketConn.WriteTo(packet[3:], spc.rAddr)
}

func (spc *ssPacketConn) ReadFrom(b []byte) (int, net.Addr, error) {
	n, _, e := spc.PacketConn.ReadFrom(b)
	if e != nil {
		return 0, nil, e
	}

	addr := socks5.SplitAddr(b[:n])
	if addr == nil {
		return 0, nil, errors.New("parse addr error")
	}

	udpAddr := addr.UDPAddr()
	if udpAddr == nil {
		return 0, nil, errors.New("parse addr error")
	}

	copy(b, b[len(addr):])
	return n - len(addr), udpAddr, e
}
