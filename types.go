package main

import (
	"context"
	"net"
)

type Database interface {
	DoesDomainExist(ctx context.Context, domain string) (bool, error)

	GetIPAddresses(ctx context.Context, domain string) ([]net.IP, error)
	PutIPAddresses(ctx context.Context, domain string, addresses []net.IP) error
	DeleteIPAddresses(ctx context.Context, domain string) error

	GetTXTValues(ctx context.Context, domain string) ([]string, error)
	PutTXTValues(ctx context.Context, domain string, values []string) error
	DeleteTXTValues(ctx context.Context, domain string) error

	GetCertificate(ctx context.Context, name string) ([]byte, error)
	PutCertificate(ctx context.Context, name string, data []byte) error
	DeleteCertificate(ctx context.Context, name string) error
}

type AlleyOopConfig struct {
	DNS dnsConfig
}

type dnsConfig struct {
	Domain      string
	NsAdmin     string
	NameServers []string
}
