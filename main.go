package main

import (
	utils "certmgrhttp01proxy/pkg"
	"log"
	"net/http"
	"regexp"
)

func main() {
	apiHostname, appsVip, err := utils.GetOCPEnvDetails()
	if err != nil {
		log.Fatalf("Error getting OCP environment details: %v\n", err)
	}
	log.Printf("OCP API: %s, APPS VIP: %s\n", apiHostname, appsVip)

	// Our backend is the APPS VIP
	backendServer := "http://" + appsVip + ":80"
	proxy, err := utils.NewReverseProxy(backendServer)
	if err != nil {
		log.Fatalf("Error creating reverse proxy: %v\n", err)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// We just want to handle requests to /.well-known/acme-challenge/*
		validPathRegex := regexp.MustCompile(`^/\.well-known/acme-challenge/.*`)
		if validPathRegex.MatchString(r.URL.Path) {
			log.Printf("Forwarding request to APPS VIP: %s\n", r.URL.Path)
			proxy.ServeHTTP(w, r)
		} else {
			http.Error(w, "Forbidden: Only /.well-known/acme-challenge/* is allowed", http.StatusForbidden)
		}
	})

	// Start the HTTP server

	log.Printf("Reverse proxy listening on :80, forwarding http01 challenges for %s to %s\n", apiHostname, backendServer)
	err = http.ListenAndServe(":80", nil)
	if err != nil {
		log.Fatalf("Error starting proxy: %v\n", err)
	}
}
