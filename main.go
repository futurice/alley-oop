package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/BurntSushi/toml"
	"github.com/futurice/alley-oop/autocert"
)

func fileExists(fname string) bool {
	_, err := os.Stat(fname)
	return err == nil
}

func getConfig(configFile string) AlleyOopConfig {
	var config AlleyOopConfig
	if _, err := toml.DecodeFile(configFile, &config); err != nil {
		fmt.Printf("Configuration file %s invalid: %s\n", configFile, err)
		os.Exit(1)
	}
	// Use a sane default/minimum value
	if config.DNS.RecordTTL < 300 {
		config.DNS.RecordTTL = 300
	}
	return config
}

func main() {
	if len(os.Args) != 2 {
		fmt.Printf("Usage: %s <config>\n", os.Args[0])
		os.Exit(1)
	}
	configFile := os.Args[1]
	if !fileExists(configFile) {
		fmt.Printf("Configuration file %s not found\n", configFile)
		os.Exit(1)
	}

	config := getConfig(configFile)

	db := FileDatabase(config.DB.Directory)
	api := NewAPI(config.Auth, db)
	handler := api.Handler

	// FIXME: We should have the host somewhere explicitly
	hostname := config.DNS.NameServers[0]
	m := autocert.Manager{
		Cache:      dbCertCache{db},
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(hostname),
	}

	cfg := &tls.Config{
		MinVersion:     tls.VersionTLS12,
		GetCertificate: m.GetCertificate,
	}
	srv := &http.Server{
		TLSConfig: cfg,
		Handler:   handler,
	}

	go func() {
		certHandler := m.HTTPHandler(nil)
		fmt.Printf("Starting server at http://localhost:80.\n")
		log.Fatal(http.ListenAndServe(":80", certHandler))
	}()

	go func() {
		startDNS(db, config.DNS)
	}()

	fmt.Printf("Starting server at http://localhost:443.\n")
	log.Fatal(srv.ListenAndServeTLS("", ""))
}
