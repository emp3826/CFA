package dns

import (
	D "github.com/miekg/dns"
)

const SystemDNSPlaceholder = "system"

var systemResolver *Resolver
var isolateHandler handler

func ServeDNSWithDefaultServer(msg *D.Msg) (*D.Msg, error) {
	if h := isolateHandler; h != nil {
		return handlerWithContext(h, msg)
	}

	return nil, D.ErrTime
}

func UpdateSystemDNS(addr []string) {
	if len(addr) == 0 {
		systemResolver = nil
	}

	ns := make([]NameServer, 0, len(addr))
	for _, d := range addr {
		ns = append(ns, NameServer{Addr: d})
	}

	systemResolver = NewResolver(Config{Main: ns})
}

func UpdateIsolateHandler(resolver *Resolver) {
	if resolver == nil {
		isolateHandler = nil

		return
	}

	isolateHandler = newHandler(resolver)
}
