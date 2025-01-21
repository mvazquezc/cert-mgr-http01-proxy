package utils

import (
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"context"

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

	//used for localtesting config, err = clientcmd.BuildConfigFromFlags("", "/path/to/kubeconfig")

	if err != nil {
		return nil, err
	}

	client, err := dynamic.NewForConfig(config)

	if err != nil {
		return nil, err
	}

	return client, err
}

func GetOCPEnvDetails() (apiHostname string, appsVIP string, err error) {
	// Get a new kubeClient and query the required objects
	// We need to know API DNS Record and APPS VIP

	client, err := newKubeClient()
	if err != nil {
		return "", "", err
	}

	// Get Ingress config for the cluster, from this object we can derive API endpoint and apps VIP
	ingressResource := schema.GroupVersionResource{Group: "config.openshift.io", Version: "v1", Resource: "ingresses"}
	ingressData, err := client.Resource(ingressResource).Get(context.TODO(), "cluster", metav1.GetOptions{})
	if err != nil {
		return "", "", err
	}
	ingressDomain, _, err := unstructured.NestedString(ingressData.Object, "spec", "domain")
	if err != nil {
		return "", "", err
	}
	apiHostname = strings.Replace(ingressDomain, "apps", "api", 1)
	// Resolve APPs hostname to get VIP
	ips, err := resolveDNSRecord("test." + ingressDomain)
	// Return only 1 IP in case there are more than 1
	appsVip := ips[0].String()
	if err != nil {
		return "", "", err
	}

	return apiHostname, appsVip, nil
}

func resolveDNSRecord(hostname string) (ips []net.IP, err error) {
	var r net.Resolver
	ips, err = r.LookupIP(context.TODO(), "ip", hostname)
	if err != nil {
		return nil, err
	}
	return ips, nil
}
