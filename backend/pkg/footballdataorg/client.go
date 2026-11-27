package footballdataorg

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// DefaultBaseURL is the API's address.
const DefaultBaseURL = "https://api.football-data.org/v4"

// ErrNoToken is returned when the client has no API key.
var ErrNoToken = errors.New("no football-data.org API key (set TD_FOOTBALL_DATA_ORG_TOKEN or football_data_org.token)")

// Client calls the API. The token is sent in a header and never logged or
// included in errors.
type Client struct {
	BaseURL   string
	Token     string
	HTTP      *http.Client
	UserAgent string
	// Retries is how many times to retry after a network error, a 5xx answer
	// or the rate limit (429).
	Retries int
	// RetryWait is the pause before retrying a network or server error.
	RetryWait time.Duration
	// MaxRateLimitWait caps how long to wait when the API says to slow down.
	MaxRateLimitWait time.Duration
}

// NewClient returns a client with sensible defaults.
func NewClient(token string) *Client {
	return &Client{
		BaseURL:          DefaultBaseURL,
		Token:            token,
		HTTP:             &http.Client{Timeout: 30 * time.Second},
		UserAgent:        "ticket-doctor-import/1.0",
		Retries:          2,
		RetryWait:        3 * time.Second,
		MaxRateLimitWait: 65 * time.Second, // the free tier's counter resets every minute
	}
}

// Fixtures returns a competition's matches with kick-off dates from from to to
// (inclusive, UTC dates). code is the API's competition code, e.g. "PL".
func (c *Client) Fixtures(ctx context.Context, code string, from, to time.Time) ([]Fixture, error) {
	if c.Token == "" {
		return nil, ErrNoToken
	}
	q := url.Values{
		"dateFrom": {from.UTC().Format(time.DateOnly)},
		"dateTo":   {to.UTC().Format(time.DateOnly)},
	}
	u := fmt.Sprintf("%s/competitions/%s/matches?%s", c.BaseURL, url.PathEscape(code), q.Encode())
	body, err := c.get(ctx, u)
	if err != nil {
		return nil, fmt.Errorf("fixtures for %s: %w", code, err)
	}
	return Parse(body)
}

func (c *Client) get(ctx context.Context, u string) (io.Reader, error) {
	var lastErr error
	for attempt := 0; attempt <= c.Retries; attempt++ {
		data, wait, err := c.try(ctx, u)
		if err == nil {
			return data, nil
		}
		lastErr = err
		if wait < 0 || attempt == c.Retries {
			break
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(wait):
		}
	}
	return nil, lastErr
}

// try makes one request. wait is how long to pause before retrying, or -1 if
// retrying can't help.
func (c *Client) try(ctx context.Context, u string) (io.Reader, time.Duration, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, -1, err
	}
	req.Header.Set("X-Auth-Token", c.Token)
	req.Header.Set("User-Agent", c.UserAgent)
	res, err := c.HTTP.Do(req)
	if err != nil {
		// The URL holds no secret (the token is a header), so the error is safe to show.
		return nil, c.RetryWait, err
	}
	defer res.Body.Close()

	switch {
	case res.StatusCode == http.StatusOK:
	case res.StatusCode == http.StatusUnauthorized || res.StatusCode == http.StatusForbidden:
		return nil, -1, fmt.Errorf("API key rejected (%s): check the token, and that the free plan covers this competition", res.Status)
	case res.StatusCode == http.StatusTooManyRequests:
		return nil, c.rateLimitWait(res.Header), errors.New("rate limit reached (429)")
	case res.StatusCode >= 500:
		return nil, c.RetryWait, fmt.Errorf("server error %s", res.Status)
	default:
		return nil, -1, fmt.Errorf("unexpected answer %s", res.Status)
	}
	// A fortnight of one league is a few tens of KB; cap reads in case of a wrong URL.
	data, err := io.ReadAll(io.LimitReader(res.Body, 5<<20))
	if err != nil {
		return nil, c.RetryWait, err
	}
	return bytes.NewReader(data), 0, nil
}

// rateLimitWait reads X-RequestCounter-Reset (seconds until the counter
// resets), capped at MaxRateLimitWait.
func (c *Client) rateLimitWait(h http.Header) time.Duration {
	wait := c.MaxRateLimitWait
	if s, err := strconv.Atoi(h.Get("X-RequestCounter-Reset")); err == nil && s >= 0 {
		if d := time.Duration(s+1) * time.Second; d < wait {
			wait = d
		}
	}
	return wait
}
