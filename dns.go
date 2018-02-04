package main

import (
	"fmt"
	"log"
	"net"

	"github.com/miekg/dns"
)

const SOAString string = "@ SOA prisoner.iana.org. hostmaster.root-servers.org. 2002040800 1800 900 0604800 604800"

func getMockARecord(domain string) dns.RR {
	return &dns.A{
		Hdr: dns.RR_Header{Name: domain, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 300},
		A:   net.ParseIP("127.0.0.1"),
	}
}

func getMockAAAARecord(domain string) dns.RR {
	return &dns.AAAA{
		Hdr:  dns.RR_Header{Name: domain, Rrtype: dns.TypeAAAA, Class: dns.ClassINET, Ttl: 300},
		AAAA: net.ParseIP("::1"),
	}
}

func processQuery(msg *dns.Msg, ns []dns.RR) {
	// Multiple questions are never used in practice
	q := msg.Question[0]

	switch q.Qtype {
	case dns.TypeA:
		msg.Authoritative = true
		msg.Ns = ns
		msg.Answer = append(msg.Answer, getMockARecord(q.Name))
	case dns.TypeAAAA:
		msg.Authoritative = true
		msg.Ns = ns
		msg.Answer = append(msg.Answer, getMockAAAARecord(q.Name))
	default:
		// FIXME: Add RcodeNameError response with SOA in authority section
	}
}

func getHandler(subdomain string, nameservers []string) func(dns.ResponseWriter, *dns.Msg) {
	nshdr := dns.RR_Header{Name: subdomain, Rrtype: dns.TypeNS, Class: dns.ClassINET, Ttl: 3600}

	var nsrr []dns.RR
	for _, ns := range nameservers {
		rr := new(dns.NS)
		rr.Hdr = nshdr
		rr.Ns = dns.Fqdn(ns)
		nsrr = append(nsrr, rr)
	}

	return func(w dns.ResponseWriter, req *dns.Msg) {
		msg := new(dns.Msg)
		msg.SetReply(req)

		if req.Opcode == dns.OpcodeQuery {
			processQuery(msg, nsrr)
		}
		w.WriteMsg(msg)
	}
}

func startDNS(subdomain string, nameservers []string) {
	domain := dns.Fqdn(subdomain)

	dns.HandleFunc(domain, getHandler(domain, nameservers))
	server := &dns.Server{Addr: ":53", Net: "udp"}

	fmt.Printf("Starting DNS server at localhost:53\n")
	log.Fatal(server.ListenAndServe())
}
