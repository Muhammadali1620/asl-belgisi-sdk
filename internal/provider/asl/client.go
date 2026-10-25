package asl

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// ASL server addresses (doc 1.3).
const (
	ProdURL  = "https://xtrace.aslbelgisi.uz"
	StageURL = "https://xtrace.stage.aslbelgisi.uz"
)

// Default client settings.
const (
	DefaultTimeout    = 30 * time.Second
	DefaultMaxRetries = 2
	DefaultRetryWait  = 500 * time.Millisecond
	DefaultUserAgent  = "asl-go-provider/1.0"
)

// Signer signs a document JSON before sending (E-IMZO CMS is made outside of this provider).
type Signer interface {
	Sign(ctx context.Context, documentJSON []byte) (string, error)
}

// SignerFunc lets a plain function be used as Signer.
type SignerFunc func(ctx context.Context, documentJSON []byte) (string, error)

// Sign calls the function.
func (f SignerFunc) Sign(ctx context.Context, documentJSON []byte) (string, error) {
	// Simple adapter, like http.HandlerFunc.
	return f(ctx, documentJSON)
}

// Config is the client setup; only BaseURL and Token are needed for most cases.
type Config struct {
	BaseURL    string        // ProdURL or StageURL
	Token      string        // business API key or technical user access token
	Timeout    time.Duration // time limit for one HTTP try; default 30s
	MaxRetries int           // extra tries for safe GET requests; default 2, -1 turns retries off
	RetryWait  time.Duration // first wait before retry, it grows two times each try; default 0.5s
	HTTPClient *http.Client  // own client (for tests or proxy); default is HTTP/2 client
	Signer     Signer        // optional, signs documents; without it documents go unsigned
	UserAgent  string
}

// Client is the ASL provider. It is safe for use from many goroutines.
// It keeps one HTTP/2 connection pool, so create it once and reuse it.
type Client struct {
	baseURL    *url.URL
	httpClient *http.Client
	timeout    time.Duration
	maxRetries int
	retryWait  time.Duration
	signer     Signer
	userAgent  string

	mu    sync.RWMutex
	token string
}

// New creates a client and checks the config.
func New(cfg Config) (*Client, error) {
	// Fills defaults and builds an HTTP/2 transport when no HTTP client is given.
	base, err := parseBaseURL(cfg.BaseURL)
	if err != nil {
		return nil, err
	}
	client := &Client{
		baseURL:    base,
		httpClient: buildHTTPClient(cfg.HTTPClient),
		timeout:    valueOr(cfg.Timeout, DefaultTimeout),
		maxRetries: cfg.MaxRetries,
		retryWait:  valueOr(cfg.RetryWait, DefaultRetryWait),
		signer:     cfg.Signer,
		userAgent:  cfg.UserAgent,
		token:      strings.TrimSpace(cfg.Token),
	}
	if client.maxRetries == 0 {
		client.maxRetries = DefaultMaxRetries
	}
	if client.maxRetries < 0 {
		client.maxRetries = 0
	}
	if client.userAgent == "" {
		client.userAgent = DefaultUserAgent
	}
	return client, nil
}

// SetToken changes the bearer token used in next requests.
func (c *Client) SetToken(token string) {
	// Safe to call while other requests run.
	c.mu.Lock()
	defer c.mu.Unlock()
	c.token = strings.TrimSpace(token)
}

// Token returns the current bearer token.
func (c *Client) Token() string {
	// Read lock, so many requests can read it at the same time.
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.token
}

// parseBaseURL checks that the base URL is an absolute http(s) URL.
func parseBaseURL(raw string) (*url.URL, error) {
	// A trailing slash is removed, so paths can be joined safely.
	raw = strings.TrimRight(strings.TrimSpace(raw), "/")
	if raw == "" {
		return nil, invalidArgument("BaseURL", "base URL is empty, use asl.ProdURL or asl.StageURL")
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "https" && parsed.Scheme != "http") {
		return nil, invalidArgument("BaseURL", "base URL %q is not a valid http(s) URL", raw)
	}
	return parsed, nil
}

// buildHTTPClient copies the given client or creates a new HTTP/2 client; redirects are not followed.
func buildHTTPClient(custom *http.Client) *http.Client {
	// We need to see 307 answers ourselves (private codes method), so redirects are turned off.
	var client http.Client
	if custom != nil {
		client = *custom
	} else {
		client = http.Client{Transport: NewTransport()}
	}
	client.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}
	return &client
}

// NewTransport creates the default transport: HTTP/2 over TLS with HTTP/1.1 fallback.
// Use it to build your own Config.HTTPClient (for example with a proxy or own certificates).
func NewTransport() *http.Transport {
	// Pings find dead HTTP/2 connections, so a broken connection is not reused for long.
	protocols := new(http.Protocols)
	protocols.SetHTTP1(true)
	protocols.SetHTTP2(true)
	dialer := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
	return &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           dialer.DialContext,
		TLSClientConfig:       &tls.Config{MinVersion: tls.VersionTLS12},
		TLSHandshakeTimeout:   10 * time.Second,
		ForceAttemptHTTP2:     true,
		Protocols:             protocols,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   20,
		IdleConnTimeout:       90 * time.Second,
		ExpectContinueTimeout: time.Second,
		HTTP2: &http.HTTP2Config{
			SendPingTimeout: 30 * time.Second,
			PingTimeout:     15 * time.Second,
		},
	}
}

// valueOr returns value, or fallback when value is not positive.
func valueOr(value, fallback time.Duration) time.Duration {
	// Small helper for duration defaults.
	if value <= 0 {
		return fallback
	}
	return value
}
