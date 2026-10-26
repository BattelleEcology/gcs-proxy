package handlers

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"regexp"
	"time"

	"cloud.google.com/go/storage"
	"github.com/kelseyhightower/envconfig"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/option"
	ghttp "google.golang.org/api/transport/http"
)

// Config represents the gcs-helper configuration that is loaded from the
// environment.
type Config struct {
	Listen     string `default:":8080"`
	BucketName string `envconfig:"BUCKET_NAME" required:"true"`
	LogLevel   string `envconfig:"LOG_LEVEL" default:"info"`
	Client     ClientConfig
	Map        MapConfig
	Proxy      ProxyConfig
}

func (c Config) Logger() *logrus.Logger {
	level, err := logrus.ParseLevel(c.LogLevel)
	if err != nil {
		level = logrus.InfoLevel
	}

	logger := logrus.New()
	logger.Out = os.Stdout
	logger.Level = level
	return logger
}

// MapConfig contains configuration for the map mode.
type MapConfig struct {
	Endpoint    string `envconfig:"GCS_HELPER_MAP_PREFIX"`
	RegexFilter string `envconfig:"GCS_HELPER_MAP_REGEX_FILTER"`
}

// ProxyConfig contains configuration for the proxy mode.
type ProxyConfig struct {
	Endpoint     string        `envconfig:"GCS_HELPER_PROXY_PREFIX"`
	LogHeaders   []string      `envconfig:"GCS_HELPER_PROXY_LOG_HEADERS"`
	Timeout      time.Duration `envconfig:"GCS_HELPER_PROXY_TIMEOUT" default:"10s"`
	BucketOnPath bool          `envconfig:"GCS_HELPER_PROXY_BUCKET_ON_PATH"`
	RootRedirect string        `envconfig:"GCS_HELPER_ROOT_REDIRECT"`

	// AllowedBuckets lists the buckets, besides BucketName, that may be
	// requested when BucketOnPath is set.
	AllowedBuckets []string `envconfig:"GCS_HELPER_PROXY_ALLOWED_BUCKETS"`
}

// ClientConfig contains configuration for the GCS client communication.
//
// It contains options related to timeouts and keep-alive connections.
type ClientConfig struct {
	Timeout         time.Duration `envconfig:"GCS_CLIENT_TIMEOUT" default:"2s"`
	IdleConnTimeout time.Duration `envconfig:"GCS_CLIENT_IDLE_CONN_TIMEOUT" default:"120s"`
	MaxIdleConns    int           `envconfig:"GCS_CLIENT_MAX_IDLE_CONNS" default:"10"`
}

// HTTPClient returns an HTTP client with the proper authentication config
// (using Google's default application credentials) and timeouts.
//
// Timeout limits the whole client call and also each transport stage (dial,
// TLS handshake, waiting for response headers). A caller that streams
// response bodies can drop the client-level timeout and still have those
// stages limited.
func (c ClientConfig) HTTPClient() (*http.Client, error) {
	dialer := net.Dialer{Timeout: c.Timeout, KeepAlive: 30 * time.Second}
	baseTransport := http.Transport{
		DialContext:           dialer.DialContext,
		ForceAttemptHTTP2:     true,
		TLSHandshakeTimeout:   c.Timeout,
		ResponseHeaderTimeout: c.Timeout,
		IdleConnTimeout:       c.IdleConnTimeout,
		MaxIdleConns:          c.MaxIdleConns,
		// Every request goes to the same host, so the default per-host
		// limit of 2 would otherwise cap the idle pool.
		MaxIdleConnsPerHost: c.MaxIdleConns,
	}
	transport, err := ghttp.NewTransport(context.Background(), &baseTransport, option.WithScopes(storage.ScopeReadOnly))
	return &http.Client{
		Timeout:   c.Timeout,
		Transport: transport,
	}, err
}

// LoadConfig loads the configuration from environment variables.
func LoadConfig() (Config, error) {
	var c Config
	err := envconfig.Process("gcs_helper", &c)
	if err != nil {
		return c, err
	}
	if _, err = regexp.Compile(c.Map.RegexFilter); err != nil {
		return c, fmt.Errorf("invalid GCS_HELPER_MAP_REGEX_FILTER: %w", err)
	}
	return c, nil
}
