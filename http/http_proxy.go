package http

import (
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"
)

type ProxyServer struct {
	targetURL    *url.URL
	proxyType    string
	reverseProxy *httputil.ReverseProxy
	client       *http.Client
	logger       *log.Logger
}

func NewProxyServer(targetURL *url.URL, proxyType string) *ProxyServer {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: false,
		},
		MaxIdleConns:       100,
		IdleConnTimeout:    90 * time.Second,
		DisableCompression: true,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}

	logger := log.New(os.Stdout, "proxy: ", log.LstdFlags|log.Lshortfile)

	reverseProxy := httputil.NewSingleHostReverseProxy(targetURL)
	reverseProxy.ErrorLog = logger

	// Custom error handler
	reverseProxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		logger.Printf("Reverse proxy error: %v", err)
		w.WriteHeader(http.StatusBadGateway)
		fmt.Fprintf(w, "Proxy Error: %v", err)
	}

	return &ProxyServer{
		targetURL:    targetURL,
		proxyType:    proxyType,
		reverseProxy: reverseProxy,
		client:       client,
		logger:       logger,
	}
}

func (p *ProxyServer) handleForwardProxy(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodConnect {
		p.handleHTTPS(w, r)
		return
	}

	targetURL := r.URL
	if !r.URL.IsAbs() {
		// If URL is not absolute, create a new absolute URL
		targetURL = &url.URL{
			Scheme:   "http",
			Host:     r.Host,
			Path:     r.URL.Path,
			RawQuery: r.URL.RawQuery,
		}
	}

	// Create a new request
	req, err := http.NewRequest(r.Method, targetURL.String(), r.Body)
	if err != nil {
		p.logger.Printf("Error creating request: %v", err)
		http.Error(w, "Error creating request", http.StatusInternalServerError)
		return
	}

	// Copy headers
	for key, values := range r.Header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	// Send request
	resp, err := p.client.Do(req)
	if err != nil {
		p.logger.Printf("Error sending request: %v", err)
		http.Error(w, "Error sending request", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Copy response headers
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// Set status code and copy body
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func (p *ProxyServer) handleHTTPS(w http.ResponseWriter, r *http.Request) {
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		p.logger.Printf("Hijacking not supported")
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		return
	}

	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		p.logger.Printf("Hijacking error: %v", err)
		http.Error(w, "Hijacking error", http.StatusInternalServerError)
		return
	}

	targetConn, err := tls.Dial("tcp", r.Host, &tls.Config{
		InsecureSkipVerify: false,
	})
	if err != nil {
		p.logger.Printf("Error connecting to target: %v", err)
		clientConn.Close()
		return
	}

	// Send 200 OK response
	clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))

	// Start bidirectional copy
	go func() {
		io.Copy(targetConn, clientConn)
		targetConn.Close()
	}()

	go func() {
		io.Copy(clientConn, targetConn)
		clientConn.Close()
	}()
}

func (p *ProxyServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	p.logger.Printf("Received request: %s %s", r.Method, r.URL.String())

	switch p.proxyType {
	case "forward":
		p.handleForwardProxy(w, r)
	case "reverse":
		p.reverseProxy.ServeHTTP(w, r)
	}

	p.logger.Printf("Completed request in %v", time.Since(startTime))
}

func main() {
	var (
		listenAddr = flag.String("listen", ":8080", "Listen address")
		targetAddr = flag.String("target", "", "Target address (for reverse proxy)")
		proxyType  = flag.String("type", "forward", "Proxy type (forward/reverse)")
	)
	flag.Parse()

	var targetURL *url.URL
	var err error

	if *proxyType == "reverse" && *targetAddr != "" {
		if !strings.HasPrefix(*targetAddr, "http://") && !strings.HasPrefix(*targetAddr, "https://") {
			*targetAddr = "http://" + *targetAddr
		}
		targetURL, err = url.Parse(*targetAddr)
		if err != nil {
			log.Fatalf("Invalid target address: %v", err)
		}
	}

	proxy := NewProxyServer(targetURL, *proxyType)
	server := &http.Server{
		Addr:         *listenAddr,
		Handler:      proxy,
		ReadTimeout:  1 * time.Minute,
		WriteTimeout: 1 * time.Minute,
		ErrorLog:     proxy.logger,
	}

	log.Printf("Starting %s proxy server on %s", *proxyType, *listenAddr)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
