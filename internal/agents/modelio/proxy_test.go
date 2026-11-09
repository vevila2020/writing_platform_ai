package modelio

import (
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
)

func TestModelHTTPClientUsesConfiguredProxy(t *testing.T) {
	var tunnelRequests atomic.Int32
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodConnect {
			http.Error(w, "expected CONNECT tunnel", http.StatusBadRequest)
			return
		}
		tunnelRequests.Add(1)
		upstream, err := net.Dial("tcp", r.Host)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		client, _, err := w.(http.Hijacker).Hijack()
		if err != nil {
			upstream.Close()
			return
		}
		_, _ = client.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
		go func() {
			defer client.Close()
			defer upstream.Close()
			_, _ = io.Copy(client, upstream)
		}()
		_, _ = io.Copy(upstream, client)
	}))
	t.Cleanup(proxy.Close)

	proxyURL, err := url.Parse(proxy.URL)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("HTTP_PROXY", proxyURL.String())
	t.Setenv("HTTPS_PROXY", proxyURL.String())
	t.Setenv("NO_PROXY", "")

	client := modelHTTPClient()
	t.Cleanup(client.CloseIdleConnections)
	request, _ := http.NewRequest(http.MethodGet, "https://models.example.test/v1/models", nil)
	target, err := client.Transport.(*http.Transport).Proxy(request)
	if err != nil {
		t.Fatal(err)
	}
	if target == nil || target.String() != proxyURL.String() {
		t.Fatalf("proxy selection = %v, want %s", target, proxyURL)
	}

	// The host name is intentionally non-resolvable: reaching the proxy proves
	// the request follows the configured proxy instead of falling back to DNS.
	_, _ = client.Get("https://models.example.test/v1/models")
	if tunnelRequests.Load() != 1 {
		t.Fatalf("proxy CONNECT requests = %d, want 1", tunnelRequests.Load())
	}
}

func TestModelHTTPClientDirectsLocalEndpoints(t *testing.T) {
	var proxyRequests atomic.Int32
	proxy := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		proxyRequests.Add(1)
	}))
	t.Cleanup(proxy.Close)
	t.Setenv("HTTP_PROXY", proxy.URL)
	t.Setenv("HTTPS_PROXY", proxy.URL)
	t.Setenv("NO_PROXY", "localhost,127.0.0.1,::1")

	endpoint := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(endpoint.Close)
	client := modelHTTPClient()
	t.Cleanup(client.CloseIdleConnections)
	response, err := client.Get(endpoint.URL)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusNoContent || proxyRequests.Load() != 0 {
		t.Fatalf("status=%d proxy_requests=%d", response.StatusCode, proxyRequests.Load())
	}
}

func TestModelHTTPClientDirectsWithoutProxy(t *testing.T) {
	for _, key := range []string{"HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY", "NO_PROXY", "http_proxy", "https_proxy", "all_proxy", "no_proxy"} {
		t.Setenv(key, "")
	}
	client := modelHTTPClient()
	t.Cleanup(client.CloseIdleConnections)
	request, _ := http.NewRequest(http.MethodGet, "https://models.example.test/v1/models", nil)
	target, err := client.Transport.(*http.Transport).Proxy(request)
	if err != nil {
		t.Fatal(err)
	}
	if target != nil {
		t.Fatalf("proxy selection = %s, want direct", target)
	}
}
