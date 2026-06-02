package featureflag

import (
	"context"
	"fmt"

	pkgff "github.com/aureum/pkg/featureflag"
)

type Client struct {
	inner *pkgff.Client
}

func NewClient(url, appName, apiToken string) (*Client, error) {
	inner, err := pkgff.NewClient(url, appName, apiToken)
	if err != nil {
		return nil, fmt.Errorf("init feature flag client: %w", err)
	}
	return &Client{inner: inner}, nil
}

func (c *Client) IsEnabled(ctx context.Context, flag string, options ...interface{}) bool {
	return c.inner.IsEnabled(ctx, flag, options...)
}
