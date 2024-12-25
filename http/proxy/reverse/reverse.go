// proxy/reverse/proxy.go
package reverse

import (
	"net/http"
	"net/http/httputil"

	"github.com/Chandra179/go-pkg/http/proxy"
)

type Proxy struct {
	reverseProxy *httputil.ReverseProxy
	logger       proxy.Logger
}

func New(config proxy.Config) *Proxy {
	reverseProxy := httputil.NewSingleHostReverseProxy(config.Target)

	// Configure error handling
	reverseProxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		config.Logger.Error("Reverse proxy error: %v", err)
		http.Error(w, "Proxy Error", http.StatusBadGateway)
	}

	// Configure request modification if needed
	reverseProxy.ModifyResponse = func(r *http.Response) error {
		r.Header.Set("X-Proxy", "Reverse-Proxy")
		return nil
	}

	return &Proxy{
		reverseProxy: reverseProxy,
		logger:       config.Logger,
	}
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p.logger.Info("Reverse proxy handling request: %s %s", r.Method, r.URL.Path)
	p.reverseProxy.ServeHTTP(w, r)
}
