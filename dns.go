package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"github.com/miekg/dns"
)

const SOAString string = "@ SOA prisoner.iana.org. hostmaster.root-servers.org. 2002040800 1800 900 0604800 604800"

func isIPv4(addr net.IP) bool {
	return strings.Contains(addr.String(), ".")
}

func getDomain(domain string) string {
	if dns.IsFqdn(domain) {
		domain = domain[0 : len(domain)-1]
	}
	return strings.ToLower(domain)
}

func getARecords(fqdn string, recordTTL int, ipaddrs []net.IP) ([]dns.RR, error) {
	var records []dns.RR
	for _, ip := range ipaddrs {
		if !isIPv4(ip) {
			// We just skip all the IPv6 addresses
			continue
		}
		str := fmt.Sprintf("%s %d IN A %s", fqdn, recordTTL, ip.String())
		rr, err := dns.NewRR(str)
		if err != nil {
			return nil, err
		}
		records = append(records, rr)
	}
	return records, nil
}

func getAAAARecords(fqdn string, recordTTL int, ipaddrs []net.IP) ([]dns.RR, error) {
	var records []dns.RR
	for _, ip := range ipaddrs {
		if isIPv4(ip) {
			// We just skip all the IPv4 addresses
			continue
		}
		str := fmt.Sprintf("%s %d IN AAAA %s", fqdn, recordTTL, ip.String())
		rr, err := dns.NewRR(str)
		if err != nil {
			return nil, err
		}
		records = append(records, rr)
	}
	return records, nil
}

func getTXTRecords(fqdn string, recordTTL int, values []string) ([]dns.RR, error) {
	var records []dns.RR
	for _, val := range values {
		str := fmt.Sprintf("%s %d IN TXT %s", fqdn, recordTTL, val)
		rr, err := dns.NewRR(str)
		if err != nil {
			return nil, err
		}
		records = append(records, rr)
	}
	return records, nil
}

func processQuery(db Database, msg *dns.Msg, soa dns.RR, ns []dns.RR, config dnsConfig) error {
	var (
		answer []dns.RR
	)

	// Multiple questions are never used in practice
	q := msg.Question[0]

	// Use 1 second timeout for the database queries to avoid stalling
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	domain := getDomain(q.Name)
	domainExists, err := db.DoesDomainExist(ctx, domain)
	if err != nil {
		return err
	}

	recordTTL := config.RecordTTL

	if q.Qtype == dns.TypeA || q.Qtype == dns.TypeAAAA {
		ipaddrs, err := db.GetIPAddresses(ctx, domain)
		if err != nil {
			return err
		}
		if q.Qtype == dns.TypeA {
			answer, err = getARecords(q.Name, recordTTL, ipaddrs)
		} else {
			answer, err = getAAAARecords(q.Name, recordTTL, ipaddrs)
		}
		if err != nil {
			return err
		}
	} else if q.Qtype == dns.TypeTXT {
		txtvals, err := db.GetTXTValues(ctx, domain)
		if err != nil {
			return err
		}
		answer, err = getTXTRecords(q.Name, recordTTL, txtvals)
		if err != nil {
			return err
		}
	}

	if len(answer) == 0 {
		// Default response is authoritative with SOA
		msg.Authoritative = true
		msg.Ns = []dns.RR{soa}
		if !domainExists {
			// No records for the whole domain
			msg.Rcode = dns.RcodeNameError
		}
		return nil
	}

	// Send a successful response with an answer
	msg.Authoritative = true
	msg.Ns = ns
	msg.Answer = answer
	return nil
}

func getHandler(db Database, domain string, nameservers []string, config dnsConfig) func(dns.ResponseWriter, *dns.Msg) {
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
			if err := processQuery(db, msg, SOA, nsrr, config); err != nil {
				// FIXME: Handle ServFail
			}
		}
		w.WriteMsg(msg)
	}
}

func startDNS(db Database, config dnsConfig) {
	domain := dns.Fqdn(config.Domain)

	var nsfqdns []string
	for _, nsstr := range config.NameServers {
		nsfqdns = append(nsfqdns, dns.Fqdn(nsstr))
	}

	dns.HandleFunc(domain, getHandler(db, domain, nsfqdns, config))
	server := &dns.Server{Addr: ":53", Net: "udp"}

	fmt.Printf("Starting DNS server at localhost:53\n")
	log.Fatal(server.ListenAndServe())
}
