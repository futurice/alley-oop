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

func (db *MemoryDatabase) GetIPAddresses(ctx context.Context, fqdn string) ([]net.IP, error) {
	return db.ipaddrs[fqdn], nil
}

func (db *MemoryDatabase) PutIPAddresses(ctx context.Context, fqdn string, addresses []net.IP) error {
	if db.ipaddrs == nil {
		db.ipaddrs = make(map[string][]net.IP)
	}
	db.ipaddrs[fqdn] = addresses
	return nil
}

func (db *MemoryDatabase) DeleteIPAddresses(ctx context.Context, fqdn string) error {
	delete(db.ipaddrs, fqdn)
	return nil
}

func (db *MemoryDatabase) GetTXTValues(ctx context.Context, fqdn string) ([]string, error) {
	return db.txtvals[fqdn], nil
}

func (db *MemoryDatabase) PutTXTValues(ctx context.Context, fqdn string, values []string) error {
	if db.txtvals == nil {
		db.txtvals = make(map[string][]string)
	}
	db.txtvals[fqdn] = values
	return nil
}

func (db *MemoryDatabase) DeleteTXTValues(ctx context.Context, fqdn string) error {
	delete(db.txtvals, fqdn)
	return nil
}

func (db *MemoryDatabase) GetCertificate(ctx context.Context, fqdn string) ([]byte, error) {
	return db.certdata[fqdn], nil
}

func (db *MemoryDatabase) PutCertificate(ctx context.Context, fqdn string, data []byte) error {
	if db.certdata == nil {
		db.certdata = make(map[string][]byte)
	}
	db.certdata[fqdn] = data
	return nil
}

func (db *MemoryDatabase) DeleteCertificate(ctx context.Context, fqdn string) error {
	delete(db.certdata, fqdn)
	return nil
}
