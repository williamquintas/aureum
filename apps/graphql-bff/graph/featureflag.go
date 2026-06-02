package graph

import (
	"context"
	"fmt"

	"github.com/99designs/gqlgen/graphql"

	"github.com/aureum/graphql-bff/internal/infrastructure/featureflag"
)

func FeatureFlagDirective(ffClient *featureflag.Client) func(ctx context.Context, obj interface{}, next graphql.Resolver, name string) (res interface{}, err error) {
	return func(ctx context.Context, obj interface{}, next graphql.Resolver, name string) (interface{}, error) {
		if ffClient == nil || !ffClient.IsEnabled(ctx, name) {
			return nil, fmt.Errorf("feature '%s' is not enabled", name)
		}
		return next(ctx)
	}
}
