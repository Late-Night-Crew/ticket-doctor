package footballdata

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// DefaultBaseURL is where football-data.co.uk serves its files.
const DefaultBaseURL = "https://football-data.co.uk"

// Client downloads results files. The data is free; be polite: an import makes
// one request per league and season (six a day for the MVP).
type Client struct {
	BaseURL   string
	HTTP      *http.Client
	UserAgent string
	// Retries is how many times to retry after a network error or a 5xx answer.
	Retries int
	// RetryWait is the pause between attempts.
	RetryWait time.Duration
}

// NewClient returns a client with sensible defaults.
func NewClient(baseURL string) *Client {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	return &Client{
		BaseURL:   baseURL,
		HTTP:      &http.Client{Timeout: 30 * time.Second}, // follows redirects (www → bare domain)
		UserAgent: "ticket-doctor-import/1.0",
		Retries:   2,
		RetryWait: 3 * time.Second,
	}
}

// URL returns the results file address for a division and season.
func (c *Client) URL(division, season string) string {
	return fmt.Sprintf("%s/mmz4281/%s/%s.csv", c.BaseURL, season, division)
}

// Results downloads and parses one league season.
func (c *Client) Results(ctx context.Context, division, season string) ([]Match, []RowError, error) {
	if err := ValidSeason(season); err != nil {
		return nil, nil, err
	}
	body, err := c.get(ctx, c.URL(division, season))
	if err != nil {
		return nil, nil, err
	}
	return Parse(body)
}

func (c *Client) get(ctx context.Context, url string) (io.Reader, error) {
	var lastErr error
	for attempt := 0; attempt <= c.Retries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(c.RetryWait):
			}
		}
		data, retry, err := c.try(ctx, url)
		if err == nil {
			return data, nil
		}
		lastErr = err
		if !retry {
			break
		}
	}
	return nil, lastErr
}

// try makes one request. retry says whether another attempt could help.
func (c *Client) try(ctx context.Context, url string) (io.Reader, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("User-Agent", c.UserAgent)
	res, err := c.HTTP.Do(req)
	if err != nil {
		return nil, true, fmt.Errorf("download %s: %w", url, err)
	}
	defer res.Body.Close()
	switch {
	case res.StatusCode == http.StatusNotFound:
		return nil, false, fmt.Errorf("download %s: not found (season not published yet?)", url)
	case res.StatusCode >= 500:
		return nil, true, fmt.Errorf("download %s: server error %s", url, res.Status)
	case res.StatusCode != http.StatusOK:
		return nil, false, fmt.Errorf("download %s: %s", url, res.Status)
	}
	// Results files are well under 1 MB; cap reads in case of a wrong URL.
	data, err := io.ReadAll(io.LimitReader(res.Body, 10<<20))
	if err != nil {
		return nil, true, fmt.Errorf("download %s: %w", url, err)
	}
	return strings.NewReader(string(data)), false, nil
}
