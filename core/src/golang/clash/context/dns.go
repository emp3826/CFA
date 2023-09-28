package context

import (
	"github.com/gofrs/uuid"
	"github.com/miekg/dns"
)

const (
	DNSTypeHost   = "host"
	DNSTypeRaw    = "raw"
)

type DNSContext struct {
	id  uuid.UUID
	msg *dns.Msg
	Tp  string
}

func NewDNSContext(msg *dns.Msg) *DNSContext {
	id, _ := uuid.NewV4()
	return &DNSContext{
		id:  id,
		msg: msg,
	}
}
