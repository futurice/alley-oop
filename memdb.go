package main

import (
	"context"
	"net"
)

type MemoryDatabase struct {
	ipaddrs  map[string][]net.IP
	txtvals  map[string][]string
	certdata map[string][]byte
}

func (db *MemoryDatabase) DoesDomainExist(ctx context.Context, domain string) (bool, error) {
	ipaddrs, err := db.GetIPAddresses(ctx, domain)
	if err != nil {
		return false, err
	}
	txtvals, err := db.GetTXTValues(ctx, domain)
	if err != nil {
		return false, err
	}
	return (len(ipaddrs) > 0 || len(txtvals) > 0), nil
}

func (db *MemoryDatabase) GetIPAddresses(ctx context.Context, domain string) ([]net.IP, error) {
	return db.ipaddrs[domain], nil
}

func (db *MemoryDatabase) PutIPAddresses(ctx context.Context, domain string, addresses []net.IP) error {
	if db.ipaddrs == nil {
		db.ipaddrs = make(map[string][]net.IP)
	}
	db.ipaddrs[domain] = addresses
	return nil
}

func (db *MemoryDatabase) DeleteIPAddresses(ctx context.Context, domain string) error {
	delete(db.ipaddrs, domain)
	return nil
}

func (db *MemoryDatabase) GetTXTValues(ctx context.Context, domain string) ([]string, error) {
	return db.txtvals[domain], nil
}

func (db *MemoryDatabase) PutTXTValues(ctx context.Context, domain string, values []string) error {
	if db.txtvals == nil {
		db.txtvals = make(map[string][]string)
	}
	db.txtvals[domain] = values
	return nil
}

func (db *MemoryDatabase) DeleteTXTValues(ctx context.Context, domain string) error {
	delete(db.txtvals, domain)
	return nil
}

func (db *MemoryDatabase) GetCertificate(ctx context.Context, domain string) ([]byte, error) {
	return db.certdata[domain], nil
}

func (db *MemoryDatabase) PutCertificate(ctx context.Context, domain string, data []byte) error {
	if db.certdata == nil {
		db.certdata = make(map[string][]byte)
	}
	db.certdata[domain] = data
	return nil
}

func (db *MemoryDatabase) DeleteCertificate(ctx context.Context, domain string) error {
	delete(db.certdata, domain)
	return nil
}
