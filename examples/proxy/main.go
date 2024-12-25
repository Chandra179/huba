package main

import (
	"flag"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/Chandra179/go-pkg/http/proxy"
	"github.com/Chandra179/go-pkg/http/proxy/forward"
	"github.com/Chandra179/go-pkg/http/proxy/reverse"
)

type stdLogger struct {
	logger *log.Logger
}

func (l *stdLogger) Info(format string, v ...interface{})  { l.logger.Printf("INFO: "+format, v...) }
func (l *stdLogger) Error(format string, v ...interface{}) { l.logger.Printf("ERROR: "+format, v...) }

func main() {
	var (
		listenAddr = flag.String("listen", ":8080", "Listen address")
		targetAddr = flag.String("target", "", "Target address (for reverse proxy)")
		proxyType  = flag.String("type", "forward", "Proxy type (forward/reverse)")
	)
	flag.Parse()

	logger := &stdLogger{
		logger: log.New(os.Stdout, "", log.LstdFlags|log.Lshortfile),
	}

	config := proxy.Config{
		Logger: logger,
		TLSConfig: proxy.TLSConfig{
			InsecureSkipVerify: false,
		},
	}

	var handler proxy.Handler

	switch *proxyType {
	case "forward":
		handler = forward.New(config)
	case "reverse":
		if *targetAddr == "" {
			log.Fatal("Target address required for reverse proxy")
		}
		if !strings.HasPrefix(*targetAddr, "http") {
			*targetAddr = "http://" + *targetAddr
		}
		targetURL, err := url.Parse(*targetAddr)
		if err != nil {
			log.Fatalf("Invalid target address: %v", err)
		}
		config.Target = targetURL
		handler = reverse.New(config)
	default:
		log.Fatalf("Unknown proxy type: %s", *proxyType)
	}

	server := &http.Server{
		Addr:         *listenAddr,
		Handler:      handler,
		ReadTimeout:  1 * time.Minute,
		WriteTimeout: 1 * time.Minute,
		ErrorLog:     logger.logger,
	}

	log.Printf("Starting %s proxy server on %s", *proxyType, *listenAddr)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
