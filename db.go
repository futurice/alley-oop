package main

import (
	"github.com/miekg/dns"
)

var (
	records = make(map[string]map[uint16][]dns.RR)
)

func getDomainRecords(domain string) map[uint16][]dns.RR {
	return records[domain]
}

func addDomainRecord(record dns.RR) {
	// FIXME: Should check record.Hdr.Class to be dns.ClassINET
	domain := record.Header().Name
	rrtype := record.Header().Rrtype

	if records[domain] == nil {
		records[domain] = make(map[uint16][]dns.RR)
	}
	records[domain][rrtype] = append(records[domain][rrtype], record)
}
