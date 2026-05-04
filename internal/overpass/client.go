package overpass

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// DefaultEndpoints lists the public Overpass interpreters to try, primary first.
var DefaultEndpoints = []string{
	"https://overpass-api.de/api/interpreter",
	"https://overpass.kumi.systems/api/interpreter",
}

// Client posts Overpass QL queries with retry + endpoint failover.
type Client struct {
	endpoints  []string
	httpClient *http.Client
	maxRetries int
	backoff    backoffPolicy
	userAgent  string
}

type backoffPolicy struct {
	initial time.Duration
	max     time.Duration
}

// ClientOption configures NewClient.
type ClientOption func(*Client)

// WithBackoff sets the initial and capped retry backoff durations.
func WithBackoff(initial, max time.Duration) ClientOption {
	return func(c *Client) { c.backoff = backoffPolicy{initial: initial, max: max} }
}

// WithMaxRetries sets the max retry attempts per endpoint (in addition to the
// first attempt).
func WithMaxRetries(n int) ClientOption {
	return func(c *Client) { c.maxRetries = n }
}

// WithHTTPClient overrides the underlying http.Client (for custom timeouts).
func WithHTTPClient(h *http.Client) ClientOption {
	return func(c *Client) { c.httpClient = h }
}

// WithUserAgent overrides the User-Agent header.
func WithUserAgent(ua string) ClientOption {
	return func(c *Client) { c.userAgent = ua }
}

// NewClient builds a Client. endpoints[0] is tried first; on persistent
// failure the next endpoint is tried, and so on.
func NewClient(endpoints []string, opts ...ClientOption) *Client {
	c := &Client{
		endpoints: append([]string(nil), endpoints...),
		httpClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
		maxRetries: 3,
		backoff:    backoffPolicy{initial: 500 * time.Millisecond, max: 8 * time.Second},
		userAgent:  "reststop-navigator/dev (+https://github.com/tamcore/reststop-navigator)",
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Query posts q to the configured endpoints and returns the response body.
// Honours ctx for cancellation between retries.
func (c *Client) Query(ctx context.Context, q string) ([]byte, error) {
	if len(c.endpoints) == 0 {
		return nil, errors.New("overpass: no endpoints configured")
	}

	var lastErr error
	for _, ep := range c.endpoints {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		body, err := c.queryEndpoint(ctx, ep, q)
		if err == nil {
			return body, nil
		}
		lastErr = err
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil, err
		}
	}
	return nil, fmt.Errorf("overpass: all endpoints failed: %w", lastErr)
}

func (c *Client) queryEndpoint(ctx context.Context, ep, q string) ([]byte, error) {
	delay := c.backoff.initial
	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		body, retryable, err := c.doRequest(ctx, ep, q)
		if err == nil {
			return body, nil
		}
		lastErr = err
		if !retryable || attempt == c.maxRetries {
			return nil, err
		}

		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		delay *= 2
		if delay > c.backoff.max {
			delay = c.backoff.max
		}
	}
	return nil, lastErr
}

func (c *Client) doRequest(ctx context.Context, ep, q string) ([]byte, bool, error) {
	form := url.Values{"data": {q}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ep, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil, false, err
		}
		return nil, true, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, true, err
	}

	switch {
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		return body, false, nil
	case resp.StatusCode == http.StatusTooManyRequests || (resp.StatusCode >= 500 && resp.StatusCode < 600):
		return nil, true, fmt.Errorf("overpass %s: status %d", ep, resp.StatusCode)
	default:
		return nil, false, fmt.Errorf("overpass %s: status %d", ep, resp.StatusCode)
	}
}
