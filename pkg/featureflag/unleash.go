package featureflag

import (
	"context"
	"net/http"

	"github.com/Unleash/unleash-client-go/v4"
)

type Client struct {
	client *unleash.Client
}

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

func (c *Client) IsEnabled(ctx context.Context, flag string, options ...interface{}) bool {
	return c.client.IsEnabled(flag)
}

func (c *Client) Close() error {
	return c.client.Close()
}
