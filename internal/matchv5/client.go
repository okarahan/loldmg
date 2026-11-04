package matchv5

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/omerkarahan/loldmg/internal/models"
)

const baseURL = "https://%s.api.riotgames.com/lol/match/v5/matches"

// Client fetches match-v5 data from Riot's API.
type Client struct {
	region string
	apiKey string
	http   *http.Client
}

// Option customizes a Client.
type Option func(*Client)

// WithHTTPClient replaces the default HTTP client.
func WithHTTPClient(c *http.Client) Option {
	return func(cl *Client) { cl.http = c }
}

// WithRegion sets the routing region (americas, europe, asia, sea, esports).
func WithRegion(region string) Option {
	return func(cl *Client) { cl.region = region }
}

// New creates a match-v5 client. apiKey is your Riot API key.
func New(apiKey string, opts ...Option) *Client {
	c := &Client{
		apiKey: apiKey,
		region: "europe",
		http:   &http.Client{Timeout: 30 * time.Second},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *Client) url(path string) string {
	return fmt.Sprintf(baseURL+path, c.region)
}

func (c *Client) get(ctx context.Context, url string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-Riot-Token", c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("GET %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("GET %s returned %d: %s", url, resp.StatusCode, string(body))
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// GetTimeline fetches the match timeline.
func (c *Client) GetTimeline(ctx context.Context, matchID string) (*models.Timeline, error) {
	url := c.url("/" + matchID + "/timeline")
	var tl models.Timeline
	if err := c.get(ctx, url, &tl); err != nil {
		return nil, err
	}
	return &tl, nil
}

// GetMatchInfo fetches participant identities (names, champions, team).
func (c *Client) GetMatchInfo(ctx context.Context, matchID string) (*models.MatchInfo, error) {
	url := c.url("/" + matchID)
	var mi models.MatchInfo
	if err := c.get(ctx, url, &mi); err != nil {
		return nil, err
	}
	return &mi, nil
}
