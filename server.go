package main

import (
	"net/http"
	"net/url"
	"strings"

	"cloud.google.com/go/storage"
	"github.com/NYTimes/gcs-helper/v3/handlers"
)

func getHandler(c handlers.Config, client *storage.Client, hc *http.Client) http.HandlerFunc {
	proxyHandler := handlers.Proxy(c, hc)
	mapHandler := handlers.Map(c, client)

	return func(w http.ResponseWriter, r *http.Request) {
		switch {
		case stripPrefix(r.URL, c.Proxy.Endpoint):
			proxyHandler.ServeHTTP(w, r)
		case stripPrefix(r.URL, c.Map.Endpoint):
			mapHandler.ServeHTTP(w, r)
		case r.URL.Path == "/":
			// healthcheck
			w.WriteHeader(http.StatusOK)
		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
	}
}

// stripPrefix removes prefix from u's path if it matches, leaving a path that
// starts with "/". RawPath is updated too so that escaped characters in object
// names are kept.
func stripPrefix(u *url.URL, prefix string) bool {
	p, ok := cutPathPrefix(u.Path, prefix)
	if !ok {
		return false
	}
	var rawPath string
	if u.RawPath != "" {
		if rawPath, ok = cutPathPrefix(u.RawPath, prefix); !ok {
			return false
		}
	}
	u.Path, u.RawPath = p, rawPath
	return true
}

// cutPathPrefix matches prefix against whole path segments: "/proxy" matches
// "/proxy" and "/proxy/a", but not "/proxyx". An empty prefix matches
// everything.
func cutPathPrefix(p, prefix string) (string, bool) {
	rest, ok := strings.CutPrefix(p, prefix)
	if !ok {
		return "", false
	}
	if strings.HasPrefix(rest, "/") {
		return rest, true
	}
	if rest != "" && prefix != "" && !strings.HasSuffix(prefix, "/") {
		return "", false
	}
	return "/" + rest, true
}
