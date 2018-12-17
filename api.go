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

	"github.com/futurice/alley-oop/autocert"
	"github.com/julienschmidt/httprouter"
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
	w.Header().Set("Cache-Control", "no-store, must-revalidate")
	fmt.Fprintf(w, "alley-oop v1.1.0\n")
}

func (api *API) v1update(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	var (
		hostnames []string
		myips     []string
		ips       []net.IP
	)

	w.Header().Set("Cache-Control", "no-store, must-revalidate")

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

func (api *API) v1privatekey(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	err := req.ParseForm()
	if err != nil {
		http.Error(w, "parse error", http.StatusInternalServerError)
	}

	hostnames := req.Form["hostname"]
	if len(hostnames) != 1 {
		http.Error(w, "param error", http.StatusInternalServerError)
		return
	}

	hostname := hostnames[0]
	if !hostnameRegexp.MatchString(hostname) {
		http.Error(w, "regexp error", http.StatusInternalServerError)
		return
	}

	hello := &tls.ClientHelloInfo{ServerName: hostname}
	cert, err := api.certmgr.GetCertificate(hello)
	if err != nil {
		http.Error(w, "cert error", http.StatusInternalServerError)
		return
	}

	key, err := getPrivateKey(cert)
	if err != nil {
		http.Error(w, "private key error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/x-pem-file")
	w.Header().Set("Cache-Control", "no-store, must-revalidate")
	fmt.Fprintf(w, key)
}

func (api *API) v1certificate(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	err := req.ParseForm()
	if err != nil {
		http.Error(w, "parse error", http.StatusInternalServerError)
	}

	hostnames := req.Form["hostname"]
	if len(hostnames) != 1 {
		http.Error(w, "error", http.StatusInternalServerError)
		return
	}

	hostname := hostnames[0]
	if !hostnameRegexp.MatchString(hostname) {
		http.Error(w, "error", http.StatusInternalServerError)
		return
	}

	hello := &tls.ClientHelloInfo{ServerName: hostname}
	cert, err := api.certmgr.GetCertificate(hello)
	if err != nil {
		http.Error(w, "error", http.StatusInternalServerError)
		return
	}

	certs, err := getCertificates(cert)
	if err != nil {
		http.Error(w, "error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/x-pem-file")
	w.Header().Set("Cache-Control", "no-store, must-revalidate")
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

func BasicAuth(h httprouter.Handle, requiredUser, requiredPassword string) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		// Get the Basic Authentication credentials
		user, password, hasAuth := r.BasicAuth()

		if hasAuth && user == requiredUser && password == requiredPassword {
			// Delegate request to the given handle
			h(w, r, ps)
		} else {
			// Request Basic Authentication otherwise
			w.Header().Set("WWW-Authenticate", "Basic realm=Restricted")
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		}
	}
}

func NewAPI(auth authConfig, db Database) *API {
	authWrapper := func(h httprouter.Handle) httprouter.Handle {
		return BasicAuth(h, auth.Username, auth.Password)
	}

	api := &API{db: db}
	router := httprouter.New()
	router.GET("/", api.index)
	router.GET("/v1/update", authWrapper(api.v1update))
	router.GET("/v1/privatekey", authWrapper(api.v1privatekey))
	router.GET("/v1/certificate", authWrapper(api.v1certificate))
	api.Handler = router

	manager := autocert.Manager{
		Cache:  dbCertCache{db},
		Prompt: autocert.AcceptTOS,
	}
	manager.DNSHandler(dbTxtHandler{db})
	api.certmgr = manager

	return api
}
