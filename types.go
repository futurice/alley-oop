package main

import (
	"context"
	"net"
)

type Database interface {
	GetIPAddresses(ctx context.Context, fqdn string) ([]net.IP, error)
	PutIPAddresses(ctx context.Context, fqdn string, addresses []net.IP) error
	DeleteIPAddresses(ctx context.Context, fqdn string) error

	GetTXTValues(ctx context.Context, fqdn string) ([]string, error)
	PutTXTValues(ctx context.Context, fqdn string, values []string) error
	DeleteTXTValues(ctx context.Context, fqdn string) error

	GetCertificate(ctx context.Context, fqdn string) ([]byte, error)
	PutCertificate(ctx context.Context, fqdn string, data []byte) error
	DeleteCertificate(ctx context.Context, fqdn string) error
}

type AlleyOopConfig struct {
	DNS dnsConfig
}

type dnsConfig struct {
	Domain      string
	NsAdmin     string
	NameServers []string
}
