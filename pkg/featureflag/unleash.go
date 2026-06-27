// Package featureflag provides a client wrapper around Unleash for feature flag evaluation.
package featureflag

import (
	"context"
	"net/http"

	"github.com/Unleash/unleash-client-go/v4"
)

// Client wraps the Unleash client for feature flag evaluation.
type Client struct {
	client *unleash.Client
}

// NewClient creates a new Unleash feature flag client.
func NewClient(url, appName, apiToken string) (*Client, error) {
	c, err := unleash.NewClient(
		unleash.WithUrl(url),
		unleash.WithAppName(appName),
		unleash.WithCustomHeaders(http.Header{
			"Authorization": []string{apiToken},
		}),
	)
	if err != nil {
		return nil, err
	}
	return &Client{client: c}, nil
}

// IsEnabled checks whether a feature flag is enabled for the current context.
func (c *Client) IsEnabled(ctx context.Context, flag string, options ...interface{}) bool {
	return c.client.IsEnabled(flag)
}

// Close shuts down the Unleash client and releases resources.
func (c *Client) Close() error {
	return c.client.Close()
}
