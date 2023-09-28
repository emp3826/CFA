package outbound

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"strconv"

	"github.com/Dreamacro/clash/component/dialer"
	C "github.com/Dreamacro/clash/constant"
	"github.com/Dreamacro/clash/transport/trojan"

	"golang.org/x/net/http2"
)

type Trojan struct {
	*Base
	instance *trojan.Trojan
	option   *TrojanOption

	// for gun mux
	gunTLSConfig *tls.Config
	transport    *http2.Transport
}

type TrojanOption struct {
	BasicOption
	Name           string      `proxy:"name"`
	Server         string      `proxy:"server"`
	Port           int         `proxy:"port"`
	Password       string      `proxy:"password"`
	ALPN           []string    `proxy:"alpn,omitempty"`
	SNI            string      `proxy:"sni,omitempty"`
	SkipCertVerify bool        `proxy:"skip-cert-verify,omitempty"`
	UDP            bool        `proxy:"udp,omitempty"`
	Network        string      `proxy:"network,omitempty"`
	WSOpts         WSOptions   `proxy:"ws-opts,omitempty"`
}

func (t *Trojan) plainStream(c net.Conn) (net.Conn, error) {
	if t.option.Network == "ws" {
		host, port, _ := net.SplitHostPort(t.addr)
		wsOpts := &trojan.WebsocketOption{
			Host: host,
			Port: port,
			Path: t.option.WSOpts.Path,
		}

		if t.option.SNI != "" {
			wsOpts.Host = t.option.SNI
		}

		if len(t.option.WSOpts.Headers) != 0 {
			header := http.Header{}
			for key, value := range t.option.WSOpts.Headers {
				header.Add(key, value)
			}
			wsOpts.Headers = header
		}

		return t.instance.StreamWebsocketConn(c, wsOpts)
	}

	return t.instance.StreamConn(c)
}

// StreamConn implements C.ProxyAdapter
func (t *Trojan) StreamConn(c net.Conn, metadata *C.Metadata) (net.Conn, error) {
	var err error
	c, err = t.plainStream(c)

	if err != nil {
		return nil, err
	}

	err = t.instance.WriteHeader(c, trojan.CommandTCP, serializesSocksAddr(metadata))
	return c, err
}

// DialContext implements C.ProxyAdapter
func (t *Trojan) DialContext(ctx context.Context, metadata *C.Metadata) (_ C.Conn, err error) {
	c, err := dialer.DialContext(ctx, "tcp", t.addr)
	if err != nil {
		return nil, err
	}

	defer func(c net.Conn, err error) {
		if err != nil {
			c.Close()
		}
	}(c, err)

	c, err = t.StreamConn(c, metadata)
	if err != nil {
		return nil, err
	}

	return NewConn(c, t), err
}

// ListenPacketContext implements C.ProxyAdapter
func (t *Trojan) ListenPacketContext(ctx context.Context, metadata *C.Metadata) (_ C.PacketConn, err error) {
	var c net.Conn
	c, err = dialer.DialContext(ctx, "tcp", t.addr)
	if err != nil {
		return nil, err
	}
	defer func(c net.Conn, err error) {
		if err != nil {
			c.Close()
		}
	}(c, err)
	c, err = t.plainStream(c)
	if err != nil {
		return nil, err
	}

	err = t.instance.WriteHeader(c, trojan.CommandUDP, serializesSocksAddr(metadata))
	if err != nil {
		return nil, err
	}

	pc := t.instance.PacketConn(c)
	return newPacketConn(pc, t), err
}

func NewTrojan(option TrojanOption) (*Trojan, error) {
	addr := net.JoinHostPort(option.Server, strconv.Itoa(option.Port))

	tOption := &trojan.Option{
		Password:       option.Password,
		ALPN:           option.ALPN,
		ServerName:     option.Server,
		SkipCertVerify: option.SkipCertVerify,
	}

	if option.SNI != "" {
		tOption.ServerName = option.SNI
	}

	t := &Trojan{
		Base: &Base{
			name:  option.Name,
			addr:  addr,
			tp:    C.Trojan,
			udp:   option.UDP,
		},
		instance: trojan.New(tOption),
		option:   &option,
	}

	return t, nil
}
