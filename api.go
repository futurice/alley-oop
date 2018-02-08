package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/julienschmidt/httprouter"
)

type API struct {
	Handler http.Handler
	db      Database
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

func NewAPI(db Database) *API {
	api := &API{db: db}
	router := httprouter.New()
	router.GET("/", api.index)
	router.GET("/v1/update", api.v1update)
	api.Handler = router
	return api
}
