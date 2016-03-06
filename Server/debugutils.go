package main

import (
	"net/http"
	"net/http/httputil"
	"log"
)


func dumpHttpRequest(r *http.Request) {
	log.Printf("dumpHttpRequest(): ")
	dump, err := httputil.DumpRequest(r, true)
	if err != nil {
		log.Printf("dumpHttpRequest: error dumping http request")
		return
	}

	log.Printf("%q", dump)
}
