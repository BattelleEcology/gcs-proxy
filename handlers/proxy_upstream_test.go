package handlers

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"
)

// redirectTransport sends every request to target, keeping the path and
// query, and records the last request it saw.
type redirectTransport struct {
	target string
	err    error
	last   *http.Request
}

func (rt *redirectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	rt.last = req
	if rt.err != nil {
		return nil, rt.err
	}
	u, err := url.Parse(rt.target)
	if err != nil {
		return nil, err
	}
	out := req.Clone(req.Context())
	out.URL.Scheme, out.URL.Host, out.Host = u.Scheme, u.Host, ""
	return http.DefaultTransport.RoundTrip(out)
}

func stubProxy(t *testing.T, cfg Config, clientTimeout time.Duration, upstream http.HandlerFunc) (*proxyHandler, *redirectTransport, *bytes.Buffer) {
	t.Helper()
	server := httptest.NewServer(upstream)
	t.Cleanup(server.Close)
	rt := &redirectTransport{target: server.URL}
	h := Proxy(cfg, &http.Client{Timeout: clientTimeout, Transport: rt}).(*proxyHandler)
	var logs bytes.Buffer
	h.logger.Out = &logs
	return h, rt, &logs
}

func TestProxyStreamsPastClientTimeout(t *testing.T) {
	const chunks = 4
	h, _, _ := stubProxy(t, Config{BucketName: "b", Proxy: ProxyConfig{Timeout: 5 * time.Second}}, 100*time.Millisecond,
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			for i := 0; i < chunks; i++ {
				w.Write([]byte("x"))
				w.(http.Flusher).Flush()
				time.Sleep(60 * time.Millisecond)
			}
		})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/obj", nil))
	if got := rec.Body.String(); got != strings.Repeat("x", chunks) {
		t.Errorf("body truncated\nwant %q\ngot  %q", strings.Repeat("x", chunks), got)
	}
}

func TestProxyUpstreamErrors(t *testing.T) {
	t.Run("timeout", func(t *testing.T) {
		h, _, _ := stubProxy(t, Config{BucketName: "b", Proxy: ProxyConfig{Timeout: 50 * time.Millisecond}}, time.Second,
			func(w http.ResponseWriter, r *http.Request) {
				select {
				case <-r.Context().Done():
				case <-time.After(time.Second):
				}
			})
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/obj", nil))
		if rec.Code != http.StatusGatewayTimeout {
			t.Errorf("wrong status code\nwant %d\ngot  %d", http.StatusGatewayTimeout, rec.Code)
		}
	})
	t.Run("connection error", func(t *testing.T) {
		h, rt, _ := stubProxy(t, Config{BucketName: "b", Proxy: ProxyConfig{Timeout: time.Second}}, time.Second, nil)
		rt.err = errors.New("dial tcp: connection refused to secret-host")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/obj", nil))
		if rec.Code != http.StatusBadGateway {
			t.Errorf("wrong status code\nwant %d\ngot  %d", http.StatusBadGateway, rec.Code)
		}
		if strings.Contains(rec.Body.String(), "secret-host") {
			t.Errorf("upstream error leaked to client: %q", rec.Body.String())
		}
	})
}

func TestProxyFiltersRequest(t *testing.T) {
	h, rt, _ := stubProxy(t, Config{BucketName: "b", Proxy: ProxyConfig{Timeout: time.Second}}, time.Second,
		func(w http.ResponseWriter, r *http.Request) {})
	req := httptest.NewRequest(http.MethodGet, "/dir/obj?generation=3&acl", nil)
	req.Header.Set("Range", "bytes=0-1")
	req.Header.Set("Authorization", "Bearer caller-token")
	req.Header.Set("X-Goog-User-Project", "someone-else")
	h.ServeHTTP(httptest.NewRecorder(), req)

	if got := rt.last.URL.String(); got != "https://b.storage.googleapis.com/dir/obj?generation=3" {
		t.Errorf("wrong upstream URL: %s", got)
	}
	want := http.Header{"Range": {"bytes=0-1"}}
	if !reflect.DeepEqual(rt.last.Header, want) {
		t.Errorf("wrong upstream headers\nwant %v\ngot  %v", want, rt.last.Header)
	}
}

func TestProxyLogsStatusOfErrors(t *testing.T) {
	h, _, logs := stubProxy(t, Config{BucketName: "b", LogLevel: "debug", Proxy: ProxyConfig{Timeout: time.Second}}, time.Second, nil)
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/obj", nil))
	if !strings.Contains(logs.String(), "response=405") {
		t.Errorf("log doesn't record the 405 response: %s", logs.String())
	}
}

func TestCopyResponseHeader(t *testing.T) {
	src := http.Header{
		"Connection":     {"close, X-Private"},
		"X-Private":      {"1"},
		"Keep-Alive":     {"timeout=5"},
		"Content-Length": {"10"},
		"Etag":           {`"abc"`},
	}
	dst := http.Header{}
	copyResponseHeader(dst, src)
	want := http.Header{"Content-Length": {"10"}, "Etag": {`"abc"`}}
	if !reflect.DeepEqual(dst, want) {
		t.Errorf("wrong headers copied\nwant %v\ngot  %v", want, dst)
	}
}
