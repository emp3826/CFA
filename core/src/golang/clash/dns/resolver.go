package dns

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"strings"
	"time"

	"github.com/Dreamacro/clash/common/cache"
	"github.com/Dreamacro/clash/common/picker"
	"github.com/Dreamacro/clash/component/resolver"
	"github.com/Dreamacro/clash/component/trie"

	D "github.com/miekg/dns"
	"golang.org/x/sync/singleflight"
)

type dnsClient interface {
	Exchange(m *D.Msg) (msg *D.Msg, err error)
	ExchangeContext(ctx context.Context, m *D.Msg) (msg *D.Msg, err error)
}

type result struct {
	Msg   *D.Msg
	Error error
}

type Resolver struct {
	hosts                 *trie.DomainTrie
	main                  []dnsClient
	fallback              []dnsClient
	fallbackDomainFilters []fallbackDomainFilter
	fallbackIPFilters     []fallbackIPFilter
	group                 singleflight.Group
	lruCache              *cache.LruCache
}

// ResolveIP request with TypeA and TypeAAAA, priority return TypeA
func (r *Resolver) ResolveIP(host string) (ip net.IP, err error) {
	ch := make(chan net.IP, 1)
	go func() {
		defer close(ch)
		ip, err := r.resolveIP(host, D.TypeAAAA)
		if err != nil {
			return
		}
		ch <- ip
	}()

	ip, err = r.resolveIP(host, D.TypeA)
	if err == nil {
		return
	}

	ip, open := <-ch
	if !open {
		return nil, resolver.ErrIPNotFound
	}

	return ip, nil
}

// ResolveIPv4 request with TypeA
func (r *Resolver) ResolveIPv4(host string) (ip net.IP, err error) {
	return r.resolveIP(host, D.TypeA)
}

// ResolveIPv6 request with TypeAAAA
func (r *Resolver) ResolveIPv6(host string) (ip net.IP, err error) {
	return r.resolveIP(host, D.TypeAAAA)
}

func (r *Resolver) shouldIPFallback(ip net.IP) bool {
	for _, filter := range r.fallbackIPFilters {
		if filter.Match(ip) {
			return true
		}
	}
	return false
}

// Exchange a batch of dns request, and it use cache
func (r *Resolver) Exchange(m *D.Msg) (msg *D.Msg, err error) {
	return r.ExchangeContext(context.Background(), m)
}

// ExchangeContext a batch of dns request with context.Context, and it use cache
func (r *Resolver) ExchangeContext(ctx context.Context, m *D.Msg) (msg *D.Msg, err error) {
	if len(m.Question) == 0 {
		return nil, errors.New("should have one question at least")
	}

	q := m.Question[0]
	cache, expireTime, hit := r.lruCache.GetWithExpire(q.String())
	if hit {
		now := time.Now()
		msg = cache.(*D.Msg).Copy()
		if expireTime.Before(now) {
			setMsgTTL(msg, uint32(1)) // Continue fetch
			go r.exchangeWithoutCache(ctx, m)
		} else {
			setMsgTTL(msg, uint32(time.Until(expireTime).Seconds()))
		}
		return
	}
	return r.exchangeWithoutCache(ctx, m)
}

// ExchangeWithoutCache a batch of dns request, and it do NOT GET from cache
func (r *Resolver) exchangeWithoutCache(ctx context.Context, m *D.Msg) (msg *D.Msg, err error) {
	q := m.Question[0]

	ret, err, shared := r.group.Do(q.String(), func() (result any, err error) {
		defer func() {
			if err != nil {
				return
			}
		}()

		isIPReq := isIPRequest(q)
		if isIPReq {
			return r.ipExchange(ctx, m)
		}

		return r.batchExchange(ctx, r.main, m)
	})

	if err == nil {
		msg = ret.(*D.Msg)
		if shared {
			msg = msg.Copy()
		}
	}

	return
}

func (r *Resolver) batchExchange(ctx context.Context, clients []dnsClient, m *D.Msg) (msg *D.Msg, err error) {
	fast, ctx := picker.WithTimeout(ctx, resolver.DefaultDNSTimeout)
	for _, client := range clients {
		r := client
		fast.Go(func() (any, error) {
			m, err := r.ExchangeContext(ctx, m)
			if err != nil {
				return nil, err
			} else if m.Rcode == D.RcodeServerFailure || m.Rcode == D.RcodeRefused {
				return nil, errors.New("server failure")
			}
			return m, nil
		})
	}

	elm := fast.Wait()
	if elm == nil {
		err := errors.New("all DNS requests failed")
		if fErr := fast.Error(); fErr != nil {
			err = fmt.Errorf("%w, first error: %s", err, fErr.Error())
		}
		return nil, err
	}

	msg = elm.(*D.Msg)
	return
}

func (r *Resolver) shouldOnlyQueryFallback(m *D.Msg) bool {
	if r.fallback == nil || len(r.fallbackDomainFilters) == 0 {
		return false
	}

	domain := r.msgToDomain(m)

	if domain == "" {
		return false
	}

	for _, df := range r.fallbackDomainFilters {
		if df.Match(domain) {
			return true
		}
	}

	return false
}

func (r *Resolver) ipExchange(ctx context.Context, m *D.Msg) (msg *D.Msg, err error) {
	onlyFallback := r.shouldOnlyQueryFallback(m)

	if onlyFallback {
		res := <-r.asyncExchange(ctx, r.fallback, m)
		return res.Msg, res.Error
	}

	msgCh := r.asyncExchange(ctx, r.main, m)

	if r.fallback == nil { // directly return if no fallback servers are available
		res := <-msgCh
		msg, err = res.Msg, res.Error
		return
	}

	fallbackMsg := r.asyncExchange(ctx, r.fallback, m)
	res := <-msgCh
	if res.Error == nil {
		if ips := msgToIP(res.Msg); len(ips) != 0 {
			if !r.shouldIPFallback(ips[0]) {
				msg = res.Msg // no need to wait for fallback result
				err = res.Error
				return msg, err
			}
		}
	}

	res = <-fallbackMsg
	msg, err = res.Msg, res.Error
	return
}

func (r *Resolver) resolveIP(host string, dnsType uint16) (ip net.IP, err error) {
	ip = net.ParseIP(host)
	if ip != nil {
		isIPv4 := ip.To4() != nil
		if dnsType == D.TypeAAAA && !isIPv4 {
			return ip, nil
		} else if dnsType == D.TypeA && isIPv4 {
			return ip, nil
		} else {
			return nil, resolver.ErrIPVersion
		}
	}

	query := &D.Msg{}
	query.SetQuestion(D.Fqdn(host), dnsType)

	msg, err := r.Exchange(query)
	if err != nil {
		return nil, err
	}

	ips := msgToIP(msg)
	ipLength := len(ips)
	if ipLength == 0 {
		return nil, resolver.ErrIPNotFound
	}

	ip = ips[rand.Intn(ipLength)]
	return
}

func (r *Resolver) msgToDomain(msg *D.Msg) string {
	if len(msg.Question) > 0 {
		return strings.TrimRight(msg.Question[0].Name, ".")
	}

	return ""
}

func (r *Resolver) asyncExchange(ctx context.Context, client []dnsClient, msg *D.Msg) <-chan *result {
	ch := make(chan *result, 1)
	go func() {
		res, err := r.batchExchange(ctx, client, msg)
		ch <- &result{Msg: res, Error: err}
	}()
	return ch
}

type NameServer struct {
	Net       string
	Addr      string
	Interface string
}

type FallbackFilter struct {
	GeoIP     bool
	GeoIPCode string
	IPCIDR    []*net.IPNet
	Domain    []string
}

type Config struct {
	Main, Fallback []NameServer
	Default        []NameServer
	IPv6           bool
	FallbackFilter FallbackFilter
	Hosts          *trie.DomainTrie
}

func NewResolver(config Config) *Resolver {
	defaultResolver := &Resolver{
		main:     transform(config.Default, nil),
		lruCache: cache.NewLRUCache(cache.WithSize(4096), cache.WithStale(true)),
	}

	r := &Resolver{
		main:     transform(config.Main, defaultResolver),
		lruCache: cache.NewLRUCache(cache.WithSize(4096), cache.WithStale(true)),
		hosts:    config.Hosts,
	}

	if len(config.Fallback) != 0 {
		r.fallback = transform(config.Fallback, defaultResolver)
	}

	fallbackIPFilters := []fallbackIPFilter{}
	if config.FallbackFilter.GeoIP {
		fallbackIPFilters = append(fallbackIPFilters, &geoipFilter{
			code: config.FallbackFilter.GeoIPCode,
		})
	}
	for _, ipnet := range config.FallbackFilter.IPCIDR {
		fallbackIPFilters = append(fallbackIPFilters, &ipnetFilter{ipnet: ipnet})
	}
	r.fallbackIPFilters = fallbackIPFilters

	if len(config.FallbackFilter.Domain) != 0 {
		fallbackDomainFilters := []fallbackDomainFilter{NewDomainFilter(config.FallbackFilter.Domain)}
		r.fallbackDomainFilters = fallbackDomainFilters
	}

	return r
}
