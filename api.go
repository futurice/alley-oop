package main

import (
	"fmt"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func index(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	fmt.Fprintf(w, "Hello, world! You should be now using HTTPS!\n")
}

func getApiHandler() http.Handler {
	router := httprouter.New()
	router.GET("/", index)
	return router
}
