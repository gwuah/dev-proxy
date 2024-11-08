package internal

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httputil"
)

func NewProxy(logger *slog.Logger) *httputil.ReverseProxy {
	// reverseProxy := httputil.NewSingleHostReverseProxy(&url.URL{
	// 	Scheme: DOCKER_SCHEME,
	// 	Host:   DOCKER_LISTENER,
	// })

	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			originalURL := *req.URL

			// req.URL.Host = proxyURL.Host
			// req.URL.Scheme = proxyURL.Scheme

			// Set proxy headers
			// req.Header.Set("X-Forwarded-Host", req.Host)
			// req.Header.Set("X-Original-URL", originalURL.String())

			// Preserve the original Host header if needed
			// req.Host = req.URL.Host

			logger.Info(fmt.Sprintf("Forwarding request: %s %s -> %s", req.Method, originalURL.String(), req.URL.String()))

		},
	}
	return proxy
}
