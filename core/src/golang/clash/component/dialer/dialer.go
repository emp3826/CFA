package dialer

import (
	"context"
	"errors"
	"net"

	"github.com/Dreamacro/clash/component/resolver"
)

func DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	return dualStackDialContext(ctx, network, address)
}

func ListenPacket(ctx context.Context, network, address string) (net.PacketConn, error) {
	if DefaultSocketHook != nil {
		return listenPacketHooked(ctx, network, address)
	}

	lc := &net.ListenConfig{}

	return lc.ListenPacket(ctx, network, address)
}

func dialContext(ctx context.Context, network string, destination net.IP, port string) (net.Conn, error) {
	if DefaultSocketHook != nil {
		return dialContextHooked(ctx, network, destination, port)
	}

	dialer := &net.Dialer{}

	return dialer.DialContext(ctx, network, net.JoinHostPort(destination.String(), port))
}

func dualStackDialContext(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}

	returned := make(chan struct{})
	defer close(returned)

	type dialResult struct {
		net.Conn
		error
		resolved bool
		ipv6     bool
		done     bool
	}
	results := make(chan dialResult)
	var primary, fallback dialResult

	startRacer := func(ctx context.Context, network, host string, ipv6 bool) {
		result := dialResult{ipv6: ipv6, done: true}
		defer func() {
			select {
			case results <- result:
			case <-returned:
				if result.Conn != nil {
					result.Conn.Close()
				}
			}
		}()

		var ip net.IP
		ip, result.error = resolver.ResolveIPv4(host)
		if result.error != nil {
			return
		}
		result.resolved = true

		result.Conn, result.error = dialContext(ctx, network, ip, port)
	}

	go startRacer(ctx, network+"4", host, false)
	go startRacer(ctx, network+"6", host, true)

	for res := range results {
		if res.error == nil {
			return res.Conn, nil
		}

		if !res.ipv6 {
			primary = res
		} else {
			fallback = res
		}

		if primary.done && fallback.done {
			if primary.resolved {
				return nil, primary.error
			} else if fallback.resolved {
				return nil, fallback.error
			} else {
				return nil, primary.error
			}
		}
	}

	return nil, errors.New("never touched")
}
