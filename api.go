package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"

	"alley-oop/autocert"
	"github.com/julienschmidt/httprouter"
	"golang.org/x/crypto/acme"
)

type API struct {
	Handler http.Handler
	db      Database
	certmgr autocert.Manager
}

var (
	hostnameRegexp = regexp.MustCompile("^(([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\\-]*[a-zA-Z0-9])\\.)*([A-Za-z0-9]|[A-Za-z0-9][A-Za-z0-9\\-]*[A-Za-z0-9])$")
)

func flattenParams(params []string) []string {
	var flattened []string

	if params == nil {
		return nil
	}
	for _, param := range params {
		for _, val := range strings.Split(param, ",") {
			flattened = append(flattened, val)
		}
	}
	return flattened
}

func haveAddressesChanged(original []net.IP, updated []net.IP) bool {
	var (
		originalMap = make(map[string]bool)
		updatedMap  = make(map[string]bool)
	)

	for _, ip := range original {
		originalMap[ip.String()] = true
	}
	for _, ip := range updated {
		updatedMap[ip.String()] = true
	}
	if len(originalMap) != len(updatedMap) {
		return true
	}
	for ipstr, _ := range updatedMap {
		if !originalMap[ipstr] {
			return true
		}
	}
	return false
}

func (api *API) index(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	fmt.Fprintf(w, "Hello, world! You should be now using HTTPS!\n")
}

func (api *API) v1update(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	var (
		hostnames []string
		myips     []string
		ips       []net.IP
	)

	// Use timeout of 10 seconds, should be enough for all needed updates
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := req.ParseForm()
	if err != nil {
		goto BadRequest
	}

	hostnames = flattenParams(req.Form["hostname"])
	if hostnames == nil {
		fmt.Fprintf(w, "notfqdn")
		return
	}
	if len(hostnames) > 20 {
		fmt.Fprintf(w, "numhost")
		return
	}

	myips = flattenParams(req.Form["myip"])
	if myips == nil {
		goto BadRequest
	}

	for _, myip := range myips {
		ip := net.ParseIP(myip)
		if ip == nil {
			goto BadRequest
		} else if ip.To4() != nil || ip.To16() != nil {
			ips = append(ips, ip)
		} else {
			goto BadRequest
		}
	}

	for idx, hostname := range hostnames {
		if idx != 0 {
			fmt.Fprintf(w, "\n")
		}
		if !hostnameRegexp.MatchString(hostname) {
			fmt.Fprintf(w, "notfqdn")
			continue
		}

		domain := strings.ToLower(hostname)
		origips, err := api.db.GetIPAddresses(ctx, domain)
		if err == nil && !haveAddressesChanged(origips, ips) {
			fmt.Fprintf(w, "nochg ")
		} else {
			fmt.Fprintf(w, "good ")
		}
		err = api.db.PutIPAddresses(ctx, domain, ips)
		if err != nil {
			fmt.Fprintf(w, "dnserr")
			continue
		}

		for idx, ip := range ips {
			if idx != 0 {
				fmt.Fprintf(w, ",")
			}
			fmt.Fprintf(w, ip.String())
		}
	}
	return

BadRequest:
	fmt.Fprintf(w, "badrequest")
	return
}

func (api *API) v1certificate(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	err := req.ParseForm()
	if err != nil {
		fmt.Fprintf(w, "notfqdn")
		return
	}

	hostnames := flattenParams(req.Form["hostname"])
	if hostnames == nil || len(hostnames) != 1 {
		fmt.Fprintf(w, "notfqdn")
		return
	}
	hostname := hostnames[0]
	if !hostnameRegexp.MatchString(hostname) {
		fmt.Fprintf(w, "notfqdn")
		return
	}

	hello := &tls.ClientHelloInfo{ServerName: hostname}
	cert, err := api.certmgr.GetCertificate(hello)
	if err != nil {
		fmt.Fprintf(w, "notfqdn")
		return
	}

	key, err := getPrivateKey(cert)
	if err != nil {
		fmt.Fprintf(w, "notfqdn")
		return
	}

	certs, err := getCertificates(cert)
	if err != nil {
		fmt.Fprintf(w, "notfqdn")
		return
	}

	fmt.Fprintf(w, "private\n")
	fmt.Fprintf(w, key)
	fmt.Fprintf(w, "public\n")
	fmt.Fprintf(w, certs)
}

type dbTxtHandler struct {
	Database
}

func (db dbTxtHandler) PutTXTRecord(ctx context.Context, domain string, value string) {
	if err := db.PutTXTValues(ctx, domain, []string{value}); err != nil {
		// FIXME: Handle error
	}
}

func (db dbTxtHandler) DeleteTXTRecord(ctx context.Context, domain string) {
	if err := db.DeleteTXTValues(ctx, domain); err != nil {
		// FIXME: Handle error
	}
}

type dbCertCache struct {
	Database
}

func (db dbCertCache) Get(ctx context.Context, name string) ([]byte, error) {
	bytes, err := db.GetCertificate(ctx, name)
	if bytes == nil {
		return nil, autocert.ErrCacheMiss
	}
	return bytes, err
}

func (db dbCertCache) Put(ctx context.Context, name string, data []byte) error {
	return db.PutCertificate(ctx, name, data)
}

func (db dbCertCache) Delete(ctx context.Context, name string) error {
	return db.DeleteCertificate(ctx, name)
}

func NewAPI(db Database) *API {
	api := &API{db: db}
	router := httprouter.New()
	router.GET("/", api.index)
	router.GET("/v1/update", api.v1update)
	router.GET("/v1/certificate", api.v1certificate)
	api.Handler = router

	client := &acme.Client{DirectoryURL: "https://acme-staging.api.letsencrypt.org/directory"}
	manager := autocert.Manager{
		Client: client,
		Cache:  dbCertCache{db},
		Prompt: autocert.AcceptTOS,
	}
	manager.DNSHandler(dbTxtHandler{db})
	api.certmgr = manager

	return api
}
