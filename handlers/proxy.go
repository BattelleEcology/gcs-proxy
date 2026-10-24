package handlers

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/textproto"
	"net/url"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// forwardedHeaders are the only request headers passed on to GCS. Forwarding
// everything would let callers change how the request is authorized or billed
// (for example with Authorization or X-Goog-User-Project).
var forwardedHeaders = []string{
	"Accept",
	"Accept-Encoding",
	"If-Match",
	"If-Modified-Since",
	"If-None-Match",
	"If-Range",
	"If-Unmodified-Since",
	"Range",
}

// forwardedQueryParams are the only query parameters passed on to GCS.
var forwardedQueryParams = []string{"generation"}

// hopHeaders are connection-specific and must not be copied from the GCS
// response (RFC 9110, section 7.6.1).
var hopHeaders = []string{
	"Connection",
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Authorization",
	"Proxy-Connection",
	"Te",
	"Trailer",
	"Transfer-Encoding",
	"Upgrade",
}

type codeWrapper struct {
	code int
	http.ResponseWriter
}

func (w *codeWrapper) WriteHeader(code int) {
	w.code = code
	w.ResponseWriter.WriteHeader(code)
}

type proxyHandler struct {
	config         Config
	logger         *logrus.Logger
	hc             *http.Client
	allowedBuckets map[string]bool
}

func (h *proxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	resp := &codeWrapper{ResponseWriter: w, code: http.StatusOK}
	var err error

	defer r.Body.Close()
	defer func() {
		if err != nil || h.logger.Level >= logrus.DebugLevel {
			fields := logrus.Fields{
				"method":        r.Method,
				"elapsed":       time.Since(start).String(),
				"path":          r.URL.RequestURI(),
				"proxyEndpoint": h.config.Proxy.Endpoint,
				"response":      resp.code,
			}
			for _, header := range h.config.Proxy.LogHeaders {
				if value := r.Header.Get(header); value != "" {
					fields["ReqHeader/"+header] = value
				}
			}
			entry := h.logger.WithFields(fields)
			if err != nil {
				entry.WithError(err).Error("failed to handle request")
			} else {
				entry.Debug("finished handling request")
			}
		}
	}()

	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(resp, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// If GCS_HELPER_ROOT_REDIRECT is set, '/' should redirect to the given path
	if r.URL.Path == "/" && h.config.Proxy.RootRedirect != "" {
		r.URL.Path = h.config.Proxy.RootRedirect
		r.URL.RawPath = ""
	}

	if r.URL.Path == "/" {
		resp.WriteHeader(http.StatusOK)
		return
	}

	host := "storage.googleapis.com"
	if h.config.Proxy.BucketOnPath {
		if !h.allowedPath(r.URL.Path) {
			http.Error(resp, "not found", http.StatusNotFound)
			return
		}
	} else {
		host = h.config.BucketName + "." + host
	}

	ctx, cancel := context.WithTimeout(r.Context(), h.config.Proxy.Timeout)
	defer cancel()

	upstream := url.URL{Scheme: "https", Host: host, Path: r.URL.Path, RawPath: r.URL.RawPath}
	query := url.Values{}
	for _, name := range forwardedQueryParams {
		if values, ok := r.URL.Query()[name]; ok {
			query[name] = values
		}
	}
	upstream.RawQuery = query.Encode()

	// no support for request body, do we care? :)
	gcsReq, err := http.NewRequestWithContext(ctx, r.Method, upstream.String(), nil)
	if err != nil {
		http.Error(resp, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	for _, name := range forwardedHeaders {
		if values := r.Header.Values(name); len(values) > 0 {
			gcsReq.Header[name] = values
		}
	}
	gcsResp, err := h.hc.Do(gcsReq)
	if err != nil {
		status := http.StatusBadGateway
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			status = http.StatusGatewayTimeout
		}
		http.Error(resp, http.StatusText(status), status)
		return
	}
	defer gcsResp.Body.Close()

	copyResponseHeader(resp.Header(), gcsResp.Header)
	resp.WriteHeader(gcsResp.StatusCode)
	_, err = io.Copy(resp, gcsResp.Body)
}

// allowedPath reports whether a path-style request names an object, rather
// than a bucket, in one of the allowed buckets. Dot segments are rejected so
// that path normalization upstream can't move the request to another bucket.
func (h *proxyHandler) allowedPath(p string) bool {
	bucket, object, ok := strings.Cut(strings.TrimPrefix(p, "/"), "/")
	if !ok || object == "" || !h.allowedBuckets[bucket] {
		return false
	}
	for _, segment := range strings.Split(object, "/") {
		if segment == "." || segment == ".." {
			return false
		}
	}
	return true
}

func copyResponseHeader(dst, src http.Header) {
	skip := make(map[string]bool, len(hopHeaders))
	for _, name := range hopHeaders {
		skip[name] = true
	}
	for _, value := range src.Values("Connection") {
		for _, name := range strings.Split(value, ",") {
			if name = textproto.TrimString(name); name != "" {
				skip[http.CanonicalHeaderKey(name)] = true
			}
		}
	}
	for name, values := range src {
		if !skip[name] {
			dst[name] = append(dst[name], values...)
		}
	}
}

// Proxy returns the proxy handler.
//
// The client's own Timeout is dropped: http.Client counts reading the body
// against it, which would cut off downloads after a 200 had already been sent.
// ProxyConfig.Timeout limits the whole request instead, and the transport
// built by ClientConfig.HTTPClient still limits connecting and waiting for
// response headers.
func Proxy(c Config, hc *http.Client) http.Handler {
	logger := c.Logger()
	streaming := *hc
	streaming.Timeout = 0
	allowed := map[string]bool{c.BucketName: true}
	for _, bucket := range c.Proxy.AllowedBuckets {
		allowed[bucket] = true
	}
	return &proxyHandler{logger: logger, hc: &streaming, config: c, allowedBuckets: allowed}
}
