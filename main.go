package main

import (
	utils "certmgrhttp01proxy/pkg"
	"log"
	"net/http"
	"os"
	"regexp"
)

func main() {
	apiHostname, appsVip, apiVip, platformType, clusterVersion, err := utils.GetOCPEnvDetails()
	if err != nil {
		log.Fatalf("Error getting OCP environment details: %v\n", err)
	}
	log.Printf("OCP API: %s, API VIP: %s, APPS VIP: %s, Platform: %s, Version: %s\n", apiHostname, apiVip, appsVip, platformType, clusterVersion)

	// We only support OCP 4.17+
	err = utils.SupportedOCPVersion(clusterVersion)
	if err != nil {
		log.Fatalf("Detected non-supported version: %v\n", err)
	}

	// If APPS VIP == API VIP we do not need the proxy
	if appsVip == apiVip {
		log.Printf("API VIP and APPS VIP are equal, no proxy needed")
		os.Exit(0)
	}

	// Create NFTRules MachineConfigs
	utils.CreateNFTablesRuleMachineConfig(apiVip, "8888")
	if err != nil {
		log.Fatalf("Error creating nft rules machineconfig: %v\n", err)
	}
	log.Println("NFTables Rules MachineConfig created/updated")
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

	log.Printf("Reverse proxy listening on :8888, forwarding http01 challenges for %s to %s\n", apiHostname, backendServer)
	err = http.ListenAndServe(":8888", nil)
	if err != nil {
		log.Fatalf("Error starting proxy: %v\n", err)
	}
}
