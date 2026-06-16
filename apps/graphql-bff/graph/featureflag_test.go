package graph

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFeatureFlagDirective(t *testing.T) {
	t.Run("nil client returns error", func(t *testing.T) {
		ffFn := FeatureFlagDirective(nil)
		next := func(ctx context.Context) (interface{}, error) {
			return "success", nil
		}
		result, err := ffFn(context.Background(), nil, next, "bff-mutations-enabled")
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "feature 'bff-mutations-enabled' is not enabled")
	})
}

func TestIsFeatureEnabled_NilClient(t *testing.T) {
	r := &Resolver{FFClient: nil}
	assert.False(t, r.isFeatureEnabled(context.Background(), "any-flag"))
}
