package main

import (
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strings"

	"github.com/julienschmidt/httprouter"
)

var (
	hostnameRegexp = regexp.MustCompile("^(([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\\-]*[a-zA-Z0-9])\\.)*([A-Za-z0-9]|[A-Za-z0-9][A-Za-z0-9\\-]*[A-Za-z0-9])$")
)

func index(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	fmt.Fprintf(w, "Hello, world! You should be now using HTTPS!\n")
}

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

func v1update(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	var (
		hostnames []string
		myips     []string
		ipv4s     []net.IP
		ipv6s     []net.IP
	)

	err := req.ParseForm()
	if err != nil {
		goto BadRequest
		// FIXME: Handle form parsing error
	}

	hostnames = flattenParams(req.Form["hostname"])
	if hostnames == nil {
		fmt.Fprintf(w, "notfqdn")
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
		} else if ip.To4() != nil {
			ipv4s = append(ipv4s, ip)
		} else if ip.To16() != nil {
			ipv6s = append(ipv6s, ip)
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

		// FIXME: update IPs
		changed := true

		if changed {
			fmt.Fprintf(w, "good ")
		} else {
			fmt.Fprintf(w, "nochg ")
		}
		for idx, ip := range ipv4s {
			if idx != 0 {
				fmt.Fprintf(w, ",")
			}
			fmt.Fprintf(w, ip.String())
		}
		for idx, ip := range ipv6s {
			if len(ipv4s) != 0 || idx != 0 {
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

func getApiHandler() http.Handler {
	router := httprouter.New()
	router.GET("/", index)
	router.GET("/v1/update", v1update)
	return router
}
