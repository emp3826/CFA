package outbound

import (
	"context"
	"net"
	"strconv"

	"github.com/Dreamacro/clash/component/dialer"
	C "github.com/Dreamacro/clash/constant"
	"github.com/Dreamacro/clash/transport/shadowsocks/core"
	"github.com/Dreamacro/clash/transport/shadowsocks/shadowaead"
	"github.com/Dreamacro/clash/transport/shadowsocks/shadowstream"
	"github.com/Dreamacro/clash/transport/ssr/obfs"
	"github.com/Dreamacro/clash/transport/ssr/protocol"
)

type ShadowSocksR struct {
	*Base
	cipher   core.Cipher
	obfs     obfs.Obfs
	protocol protocol.Protocol
}

type ShadowSocksROption struct {
	BasicOption
	Name          string `proxy:"name"`
	Server        string `proxy:"server"`
	Port          int    `proxy:"port"`
	Password      string `proxy:"password"`
	Cipher        string `proxy:"cipher"`
	Obfs          string `proxy:"obfs"`
	ObfsParam     string `proxy:"obfs-param,omitempty"`
	Protocol      string `proxy:"protocol"`
	ProtocolParam string `proxy:"protocol-param,omitempty"`
	UDP           bool   `proxy:"udp,omitempty"`
}

// StreamConn implements C.ProxyAdapter
func (ssr *ShadowSocksR) StreamConn(c net.Conn, metadata *C.Metadata) (net.Conn, error) {
	c = ssr.obfs.StreamConn(c)
	c = ssr.cipher.StreamConn(c)
	var (
		iv  []byte
		err error
	)
	switch conn := c.(type) {
	case *shadowstream.Conn:
		iv, err = conn.ObtainWriteIV()
		if err != nil {
			return nil, err
		}
	case *shadowaead.Conn:
		return nil, nil
	}
	c = ssr.protocol.StreamConn(c, iv)
	_, err = c.Write(serializesSocksAddr(metadata))
	return c, err
}

// DialContext implements C.ProxyAdapter
func (ssr *ShadowSocksR) DialContext(ctx context.Context, metadata *C.Metadata) (_ C.Conn, err error) {
	c, err := dialer.DialContext(ctx, "tcp", ssr.addr)
	if err != nil {
		return nil, err
	}

	defer func(c net.Conn, err error) {
		if err != nil {
			c.Close()
		}
	}(c, err)

	c, err = ssr.StreamConn(c, metadata)
	return NewConn(c, ssr), err
}

// ListenPacketContext implements C.ProxyAdapter
func (ssr *ShadowSocksR) ListenPacketContext(ctx context.Context, metadata *C.Metadata) (C.PacketConn, error) {
	pc, err := dialer.ListenPacket(ctx, "udp", "")
	if err != nil {
		return nil, err
	}

	addr, err := resolveUDPAddr("udp", ssr.addr)
	if err != nil {
		pc.Close()
		return nil, err
	}

	pc = ssr.cipher.PacketConn(pc)
	pc = ssr.protocol.PacketConn(pc)
	return newPacketConn(&ssPacketConn{PacketConn: pc, rAddr: addr}, ssr), nil
}

func NewShadowSocksR(option ShadowSocksROption) (*ShadowSocksR, error) {
	// SSR protocol compatibility
	// https://github.com/Dreamacro/clash/pull/2056
	if option.Cipher == "none" {
		option.Cipher = "dummy"
	}

	addr := net.JoinHostPort(option.Server, strconv.Itoa(option.Port))
	cipher := option.Cipher
	password := option.Password
	coreCiph, err := core.PickCipher(cipher, nil, password)
	if err != nil {
		return nil, err
	}
	var (
		ivSize int
		key    []byte
	)

	if option.Cipher == "dummy" {
		ivSize = 0
		key = core.Kdf(option.Password, 16)
	} else {
		ciph, ok := coreCiph.(*core.StreamCipher)
		if !ok {
			return nil, nil
		}
		ivSize = ciph.IVSize()
		key = ciph.Key
	}

	obfs, obfsOverhead, err := obfs.PickObfs(option.Obfs, &obfs.Base{
		Host:   option.Server,
		Port:   option.Port,
		Key:    key,
		IVSize: ivSize,
		Param:  option.ObfsParam,
	})
	if err != nil {
		return nil, err
	}

	protocol, err := protocol.PickProtocol(option.Protocol, &protocol.Base{
		Key:      key,
		Overhead: obfsOverhead,
		Param:    option.ProtocolParam,
	})
	if err != nil {
		return nil, err
	}

	return &ShadowSocksR{
		Base: &Base{
			name:  option.Name,
			addr:  addr,
			tp:    C.ShadowsocksR,
			udp:   option.UDP,
		},
		cipher:   coreCiph,
		obfs:     obfs,
		protocol: protocol,
	}, nil
}
