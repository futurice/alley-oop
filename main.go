package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/jvah/alley-oop/autocert"
)

type HelloWorldHandler struct {
}

func (h *HelloWorldHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello, world! You should be now using HTTPS!\n")
}

func main() {
	if len(os.Args) != 2 {
		fmt.Printf("Usage: %s <domain>\n", os.Args[0])
		os.Exit(1)
	}
	domain := os.Args[1]

	mux := http.NewServeMux()
	hwh := &HelloWorldHandler{}
	mux.Handle("/", hwh)

	m := autocert.Manager{
		Cache:      autocert.DirCache("api-certs"),
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(domain),
	}
	cfg := &tls.Config{
		MinVersion:     tls.VersionTLS12,
		GetCertificate: m.GetCertificate,
	}
	srv := &http.Server{
		TLSConfig: cfg,
		Handler:   mux,
	}

	go func() {
		handler := m.HTTPHandler(nil)
		fmt.Printf("Starting server at http://localhost:80.\n")
		log.Fatal(http.ListenAndServe(":80", handler))
	}()
	fmt.Printf("Starting server at http://localhost:443.\n")
	log.Fatal(srv.ListenAndServeTLS("", ""))
}
