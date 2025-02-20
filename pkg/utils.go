package utils

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"

	"context"

	"github.com/coreos/go-iptables/iptables"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

func NewReverseProxy(targetURL string) (*httputil.ReverseProxy, error) {
	// Parse targetURL (backend server URL)
	url, err := url.Parse(targetURL)
	if err != nil {
		return nil, err
	}

	// Create proxy
	proxy := httputil.NewSingleHostReverseProxy(url)

	// Set original host header
	proxy.Director = func(req *http.Request) {
		req.URL.Scheme = url.Scheme
		req.URL.Host = url.Host
		req.Header.Set("Host", req.Host)
		req.Header.Set("X-Proxy-Server", "cert-mgt-http01-proxy")
	}
	// Add Error Handling
	proxy.ErrorHandler = func(w http.ResponseWriter, req *http.Request, err error) {
		log.Printf("Error handling request: %v\n", err)
		http.Error(w, "Backend unavailable", http.StatusBadGateway)
	}

	// In case Proxy is set, use it
	proxy.Transport = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
	}

	return proxy, nil
}

func newKubeClient() (*dynamic.DynamicClient, error) {
	var config *rest.Config
	var err error
	config, err = rest.InClusterConfig()

	//used for localtesting
	//config, err = clientcmd.BuildConfigFromFlags("", "/home/mario/.kubeconfigs/bm-cluster")

	if err != nil {
		return nil, err
	}

	client, err := dynamic.NewForConfig(config)

	if err != nil {
		return nil, err
	}

	return client, err
}

func GetOCPEnvDetails() (apiHostname string, appsVIP string, apiVIP string, platformType string, err error) {
	// Get a new kubeClient and query the required objects
	// We need to know API DNS Record and APPS VIP

	client, err := newKubeClient()
	if err != nil {
		return "", "", "", "", err
	}
	// Get Platform type, for now we only run on BM
	infrastructureResource := schema.GroupVersionResource{Group: "config.openshift.io", Version: "v1", Resource: "infrastructures"}
	infrastructureData, err := client.Resource(infrastructureResource).Get(context.TODO(), "cluster", metav1.GetOptions{})
	if err != nil {
		return "", "", "", "", err
	}
	platformType, _, err = unstructured.NestedString(infrastructureData.Object, "status", "platform")
	if err != nil {
		return "", "", "", "", err
	}
	// Get Ingress config for the cluster, from this object we can derive API endpoint and apps VIP
	ingressResource := schema.GroupVersionResource{Group: "config.openshift.io", Version: "v1", Resource: "ingresses"}
	ingressData, err := client.Resource(ingressResource).Get(context.TODO(), "cluster", metav1.GetOptions{})
	if err != nil {
		return "", "", "", "", err
	}
	ingressDomain, _, err := unstructured.NestedString(ingressData.Object, "spec", "domain")
	if err != nil {
		return "", "", "", "", err
	}
	apiHostname = strings.Replace(ingressDomain, "apps", "api", 1)
	// Resolve APPs hostname to get VIP
	ips, err := resolveDNSRecord("test." + ingressDomain)
	// Return only 1 IP in case there are more than 1
	appsVip := ips[0].String()
	if err != nil {
		return "", "", "", "", err
	}
	// Resolve API hostname to get VIP
	ips, err = resolveDNSRecord(apiHostname)
	// Return only 1 IP in case there are more than 1
	apiVip := ips[0].String()
	if err != nil {
		return "", "", "", "", err
	}

	return apiHostname, appsVip, apiVip, platformType, nil
}

func resolveDNSRecord(hostname string) (ips []net.IP, err error) {
	var r net.Resolver
	ips, err = r.LookupIP(context.TODO(), "ip", hostname)
	if err != nil {
		return nil, err
	}
	return ips, nil
}

func portIsInUse(port int) bool {
	addr := fmt.Sprintf(":%d", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return true
	}
	_ = listener.Close()
	return false
}

func GetProxyPort() int {
	proxyPorts := [4]int{6666, 7777, 8888, 9999}
	for _, port := range proxyPorts {

		if !portIsInUse(port) {
			log.Printf("Using port %d\n", port)
			return port
		}
		log.Printf("Port %d is in use or proxy cannot bind to it (capabilities?)\n", port)
	}
	return 0
}

func ConfigureIptables(apiVip string, port int) error {
	ip, err := iptables.New()
	if err != nil {
		return err
	}
	rule := []string{
		"-d", apiVip, // Destination IP
		"-p", "tcp", // Protocol
		"--dport", "80", // Incoming port
		"-j", "REDIRECT", // Action
		"--to-port", strconv.Itoa(port), // Redirect to proxy port
	}
	exists, err := ip.Exists("nat", "PREROUTING", rule...)
	if err != nil {
		return err
	}
	if !exists {
		err = ip.Append("nat", "PREROUTING", rule...)
		if err != nil {
			return err
		}
		log.Printf("Rule %s added successfully.\n", rule)
	} else {
		log.Printf("Rule %s already exists.\n", rule)
	}

	return nil
}
