package main

import (
	"fmt"
	"log"
	"net"
	"strings"
	"time"

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

func processQuery(msg *dns.Msg, soa dns.RR, ns []dns.RR) {
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
		msg.Authoritative = true
		msg.Ns = []dns.RR{soa}
	}
}

func getHandler(domain string, nameservers []string) func(dns.ResponseWriter, *dns.Msg) {
	nshdr := dns.RR_Header{Name: domain, Rrtype: dns.TypeNS, Class: dns.ClassINET, Ttl: 3600}

	var nsrr []dns.RR
	for _, ns := range nameservers {
		rr := new(dns.NS)
		rr.Hdr = nshdr
		rr.Ns = dns.Fqdn(ns)
		nsrr = append(nsrr, rr)
	}

	SOAFormat := fmt.Sprintf("%s SOA %s %s %%s 28800 7200 604800 86400", strings.ToLower(domain), strings.ToLower(nameservers[0]), "admin.domain.foo")

	return func(w dns.ResponseWriter, req *dns.Msg) {
		serial := time.Now().Format("2006010215")
		SOAString := fmt.Sprintf(SOAFormat, serial)
		SOA, err := dns.NewRR(SOAString)
		if err != nil {
			// FIXME: Handle error, should not happen
		}

		msg := new(dns.Msg)
		msg.SetReply(req)

		if req.Opcode == dns.OpcodeQuery {
			processQuery(msg, SOA, nsrr)
		}
		w.WriteMsg(msg)
	}
}

func startDNS(domainstr string, nsstrs []string) {
	domain := dns.Fqdn(domainstr)

	var nsfqdns []string
	for _, nsstr := range nsstrs {
		nsfqdns = append(nsfqdns, dns.Fqdn(nsstr))
	}

	dns.HandleFunc(domain, getHandler(domain, nsfqdns))
	server := &dns.Server{Addr: ":53", Net: "udp"}

	fmt.Printf("Starting DNS server at localhost:53\n")
	log.Fatal(server.ListenAndServe())
}
