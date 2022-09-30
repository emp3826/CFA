package main

import (
	"testing"
	"time"

	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func exchange(address, domain string, tp uint16) ([]dns.RR, error) {
	client := dns.Client{}
	query := &dns.Msg{}
	query.SetQuestion(dns.Fqdn(domain), tp)

	r, _, err := client.Exchange(query, address)
	if err != nil {
		return nil, err
	}
	return r.Answer, nil
}

func TestClash_DNS(t *testing.T) {
	basic := `
log-level: silent
dns:
  enable: true
  listen: 0.0.0.0:8553
  nameserver:
    - 119.29.29.29
`

	err := parseAndApply(basic)
	require.NoError(t, err)
	defer cleanup()

	time.Sleep(waitTime)

	rr, err := exchange("127.0.0.1:8553", "1.1.1.1.nip.io", dns.TypeA)
	assert.NoError(t, err)
	assert.NotEmptyf(t, rr, "record empty")

	record := rr[0].(*dns.A)
	assert.Equal(t, record.A.String(), "1.1.1.1")

	rr, err = exchange("127.0.0.1:8553", "2606-4700-4700--1111.sslip.io", dns.TypeAAAA)
	assert.NoError(t, err)
	assert.Empty(t, rr)
}
