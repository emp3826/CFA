package inbound

import (
	"net"
	"strconv"

	C "github.com/Dreamacro/clash/constant"
	"github.com/Dreamacro/clash/transport/socks5"
)

// PacketAdapter is a UDP Packet adapter for socks/redir/tun
type PacketAdapter struct {
	C.UDPPacket
	Metadata *C.Metadata
}

// NewPacket is PacketAdapter generator
func NewPacket(target socks5.Addr, packet C.UDPPacket, source C.Type) *PacketAdapter {
	metadata := parseSocksAddr(target)
	metadata.NetWork = C.UDP
	metadata.Type = source
	host, port, _ := net.SplitHostPort(packet.LocalAddr().String())
	ip := net.ParseIP(host)
	metadata.SrcIP = ip
	metadata.SrcPort = port

	if port, err := strconv.Atoi(metadata.DstPort); err == nil {
		metadata.RawSrcAddr = packet.LocalAddr()
		metadata.RawDstAddr = &net.UDPAddr{IP: metadata.DstIP, Port: port}
	}

	return &PacketAdapter{
		UDPPacket: packet,
		Metadata:  metadata,
	}
}
