package main

type AlleyOopConfig struct {
	DNS dnsConfig
}

type dnsConfig struct {
	Domain      string
	NsAdmin     string
	NameServers []string
}
