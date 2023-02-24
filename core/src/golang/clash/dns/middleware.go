package dns

import (
	"net"
	"strings"

	"github.com/Dreamacro/clash/component/trie"
	"github.com/Dreamacro/clash/context"

	D "github.com/miekg/dns"
)

type (
	handler    func(ctx *context.DNSContext, r *D.Msg) (*D.Msg, error)
	middleware func(next handler) handler
)

func withHosts(hosts *trie.DomainTrie) middleware {
	return func(next handler) handler {
		return func(ctx *context.DNSContext, r *D.Msg) (*D.Msg, error) {
			q := r.Question[0]

			if !isIPRequest(q) {
				return next(ctx, r)
			}

			record := hosts.Search(strings.TrimRight(q.Name, "."))
			if record == nil {
				return next(ctx, r)
			}

			ip := record.Data.(net.IP)
			msg := r.Copy()

			if v4 := ip.To4(); v4 != nil && q.Qtype == D.TypeA {
				rr := &D.A{}
				rr.Hdr = D.RR_Header{Name: q.Name, Rrtype: D.TypeA, Class: D.ClassINET, Ttl: dnsDefaultTTL}
				rr.A = v4

				msg.Answer = []D.RR{rr}
			} else if v6 := ip.To16(); v6 != nil && q.Qtype == D.TypeAAAA {
				rr := &D.AAAA{}
				rr.Hdr = D.RR_Header{Name: q.Name, Rrtype: D.TypeAAAA, Class: D.ClassINET, Ttl: dnsDefaultTTL}
				rr.AAAA = v6

				msg.Answer = []D.RR{rr}
			} else {
				return next(ctx, r)
			}

			ctx.SetType(context.DNSTypeHost)
			msg.SetRcode(r, D.RcodeSuccess)
			msg.Authoritative = true
			msg.RecursionAvailable = true

			return msg, nil
		}
	}
}

func withResolver(resolver *Resolver) handler {
	return func(ctx *context.DNSContext, r *D.Msg) (*D.Msg, error) {
		ctx.SetType(context.DNSTypeRaw)
		q := r.Question[0]

		// return a empty AAAA msg when ipv6 disabled
		if q.Qtype == D.TypeAAAA {
			return handleMsgWithEmptyAnswer(r), nil
		}

		msg, err := resolver.Exchange(r)
		if err != nil {
			return msg, err
		}
		msg.SetRcode(r, msg.Rcode)
		msg.Authoritative = true

		return msg, nil
	}
}

func compose(middlewares []middleware, endpoint handler) handler {
	length := len(middlewares)
	h := endpoint
	for i := length - 1; i >= 0; i-- {
		middleware := middlewares[i]
		h = middleware(h)
	}

	return h
}

func newHandler(resolver *Resolver) handler {
	middlewares := []middleware{}

	if resolver.hosts != nil {
		middlewares = append(middlewares, withHosts(resolver.hosts))
	}

	return compose(middlewares, withResolver(resolver))
}
