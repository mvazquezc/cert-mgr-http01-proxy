package main

import (
	utils "certmgrhttp01proxy/pkg"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
)

func main() {
	apiHostname, appsVip, apiVip, platformType, err := utils.GetOCPEnvDetails()
	if err != nil {
		log.Fatalf("Error getting OCP environment details: %v\n", err)
	}
	log.Printf("OCP API: %s, API VIP: %s, APPS VIP: %s, Platform: %s\n", apiHostname, apiVip, appsVip, platformType)
	// If APPS VIP == API VIP we do not need the proxy
	if appsVip == apiVip {
		log.Printf("API VIP and APPS VIP are equal, no proxy needed")
		os.Exit(0)
	}

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

	// Get a port
	proxyPort := utils.GetProxyPort()
	if proxyPort == 0 {
		log.Fatalf("No ports available for the proxy to use")
	}
	iptablesRule := "iptables -t nat -A PREROUTING -d " + apiVip + " -p tcp --dport 80 -j REDIRECT --to-port " + strconv.Itoa(proxyPort)
	log.Printf("The following IPTables rule must be added for the proxy to work: %s", iptablesRule)
	// Start the HTTP server
	log.Printf("Reverse proxy listening on :%d, forwarding http01 challenges for %s to %s\n", proxyPort, apiHostname, backendServer)
	addr := fmt.Sprintf(":%d", proxyPort)
	err = http.ListenAndServe(addr, nil)
	if err != nil {
		log.Fatalf("Error starting proxy: %v\n", err)
	}
}

// TODO: When creating the IPTables rule we need to use the API VIP
