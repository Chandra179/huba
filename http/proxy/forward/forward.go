package forward

import (
	"crypto/tls"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/Chandra179/go-pkg/http/proxy"
)

type Proxy struct {
	client *http.Client
	logger proxy.Logger
}

func New(config proxy.Config) *Proxy {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: config.TLSConfig.InsecureSkipVerify,
		},
		MaxIdleConns:       100,
		IdleConnTimeout:    90 * time.Second,
		DisableCompression: true,
	}

	return &Proxy{
		client: &http.Client{
			Transport: transport,
			Timeout:   30 * time.Second,
		},
		logger: config.Logger,
	}
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodConnect {
		p.handleHTTPS(w, r)
		return
	}
	p.handleHTTP(w, r)
}

func (p *Proxy) handleHTTP(w http.ResponseWriter, r *http.Request) {
	targetURL := r.URL
	if !r.URL.IsAbs() {
		targetURL = &url.URL{
			Scheme:   "http",
			Host:     r.Host,
			Path:     r.URL.Path,
			RawQuery: r.URL.RawQuery,
		}
	}

	req, err := http.NewRequest(r.Method, targetURL.String(), r.Body)
	if err != nil {
		p.logger.Error("Error creating request: %v", err)
		http.Error(w, "Error creating request", http.StatusInternalServerError)
		return
	}

	copyHeaders(req.Header, r.Header)

	resp, err := p.client.Do(req)
	if err != nil {
		p.logger.Error("Error sending request: %v", err)
		http.Error(w, "Error sending request", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	copyHeaders(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func (p *Proxy) handleHTTPS(w http.ResponseWriter, r *http.Request) {
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		p.logger.Error("Hijacking not supported")
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		return
	}

	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		p.logger.Error("Hijacking error: %v", err)
		http.Error(w, "Hijacking error", http.StatusInternalServerError)
		return
	}

	targetConn, err := tls.Dial("tcp", r.Host, &tls.Config{
		InsecureSkipVerify: false,
	})
	if err != nil {
		p.logger.Error("Error connecting to target: %v", err)
		clientConn.Close()
		return
	}

	clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))

	go transfer(targetConn, clientConn)
	go transfer(clientConn, targetConn)
}

func transfer(dst io.WriteCloser, src io.ReadCloser) {
	defer dst.Close()
	defer src.Close()
	io.Copy(dst, src)
}

func copyHeaders(dst, src http.Header) {
	for key, values := range src {
		for _, value := range values {
			dst.Add(key, value)
		}
	}
}
