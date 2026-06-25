package graph

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aureum/graphql-bff/graph/model"
	"github.com/aureum/graphql-bff/internal/infrastructure/cache"
)

// ── Test Helpers ──────────────────────────────────────────────────────────

const testRedisAddr = "localhost:6379"
const testRedisDB = 1

// newTestCache creates a Cache backed by the running Redis (Docker) on DB 1.
// It flushes DB 1 before and after the test for isolation.
func newTestCache(t *testing.T) *cache.Cache {
	t.Helper()

	// Flush the test DB first to ensure clean state
	rdb := redis.NewClient(&redis.Options{
		Addr: testRedisAddr,
		DB:   testRedisDB,
	})
	err := rdb.FlushDB(context.Background()).Err()
	if err != nil {
		t.Skipf("Redis not available (is Docker running?): %v", err)
	}
	rdb.Close()

	c, err := cache.NewRedisCache(testRedisAddr, "", testRedisDB)
	require.NoError(t, err)

	t.Cleanup(func() {
		cleanup := redis.NewClient(&redis.Options{
			Addr: testRedisAddr,
			DB:   testRedisDB,
		})
		defer cleanup.Close()
		_ = cleanup.FlushDB(context.Background())
	})

	return c
}

// validTestIncome returns a model.Income with all required enum fields set.
func validTestIncome(id, desc string) *model.Income {
	return &model.Income{
		ID:             id,
		UserID:         "user-123",
		Description:    desc,
		Source:         "test",
		IncomeType:     model.IncomeTypeFreelance,
		ReceivedDate:   time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
		ReceivedAmount: 500000,
		Status:         model.TransactionStatusCompleted,
		CreatedAt:      time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
		UpdatedAt:      time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
	}
}

// ── cachedSingle Tests ────────────────────────────────────────────────────

func TestCachedSingle_CacheHit(t *testing.T) {
	cacheStore := newTestCache(t)
	r := &Resolver{Cache: cacheStore}
	q := &queryResolver{r}
	ctx := context.Background()

	callCount := 0

	// First call — cache MISS → calls fetchFn, stores result
	var result model.Income
	err := q.cachedSingle(ctx, "test_income_hit", "id-1", &result, func() (interface{}, error) {
		callCount++
		return validTestIncome("id-1", "cached income"), nil
	})
	require.NoError(t, err)
	assert.Equal(t, 1, callCount)
	assert.Equal(t, "id-1", result.ID)
	assert.Equal(t, "cached income", result.Description)

	// Second call — cache HIT → fetchFn is NOT called, cached data returned
	result = model.Income{}
	err = q.cachedSingle(ctx, "test_income_hit", "id-1", &result, func() (interface{}, error) {
		callCount++
		return validTestIncome("id-1", "should not be called"), nil
	})
	require.NoError(t, err)
	assert.Equal(t, 1, callCount, "fetchFn should NOT be called on cache hit")
	assert.Equal(t, "id-1", result.ID)
	assert.Equal(t, "cached income", result.Description)
}

func TestCachedSingle_CacheMiss(t *testing.T) {
	cacheStore := newTestCache(t)
	r := &Resolver{Cache: cacheStore}
	q := &queryResolver{r}
	ctx := context.Background()

	callCount := 0

	var result model.Income
	err := q.cachedSingle(ctx, "test_income_miss", "id-2", &result, func() (interface{}, error) {
		callCount++
		return validTestIncome("id-2", "fresh income"), nil
	})
	require.NoError(t, err)
	assert.Equal(t, 1, callCount)
	assert.Equal(t, "id-2", result.ID)
	assert.Equal(t, "fresh income", result.Description)

	// Verify the value was actually stored in cache by making a second call
	result = model.Income{}
	err = q.cachedSingle(ctx, "test_income_miss", "id-2", &result, func() (interface{}, error) {
		callCount++
		return validTestIncome("id-2", "should not be called"), nil
	})
	require.NoError(t, err)
	assert.Equal(t, 1, callCount, "fetchFn should NOT be called — value should be cached")
	assert.Equal(t, "fresh income", result.Description)
}

// ── cachedList Tests ──────────────────────────────────────────────────────

func TestCachedList_CacheHit(t *testing.T) {
	cacheStore := newTestCache(t)
	r := &Resolver{Cache: cacheStore}
	q := &queryResolver{r}
	ctx := context.Background()

	callCount := 0

	// First call — cache MISS
	var result model.IncomeConnection
	err := q.cachedList(ctx, "test_incomes_hit", struct{}{}, &result, func() (interface{}, error) {
		callCount++
		return &model.IncomeConnection{
			Edges:      []*model.IncomeEdge{},
			TotalCount: 3,
			PageInfo:   &model.PageInfo{HasNextPage: false},
		}, nil
	})
	require.NoError(t, err)
	assert.Equal(t, 1, callCount)
	assert.Equal(t, 3, result.TotalCount)

	// Second call — cache HIT → fetchFn NOT called
	result = model.IncomeConnection{}
	err = q.cachedList(ctx, "test_incomes_hit", struct{}{}, &result, func() (interface{}, error) {
		callCount++
		return &model.IncomeConnection{
			Edges:      []*model.IncomeEdge{{Node: validTestIncome("wrong", "should not be called")}},
			TotalCount: 999,
		}, nil
	})
	require.NoError(t, err)
	assert.Equal(t, 1, callCount, "fetchFn should NOT be called on cache hit")
	assert.Equal(t, 3, result.TotalCount)
}

func TestCachedList_CacheMiss(t *testing.T) {
	cacheStore := newTestCache(t)
	r := &Resolver{Cache: cacheStore}
	q := &queryResolver{r}
	ctx := context.Background()

	callCount := 0

	var result model.IncomeConnection
	err := q.cachedList(ctx, "test_incomes_miss", struct{}{}, &result, func() (interface{}, error) {
		callCount++
		return &model.IncomeConnection{
			Edges:      []*model.IncomeEdge{},
			TotalCount: 5,
			PageInfo:   &model.PageInfo{HasNextPage: false},
		}, nil
	})
	require.NoError(t, err)
	assert.Equal(t, 1, callCount)
	assert.Equal(t, 5, result.TotalCount)

	// Verify the value was cached
	result = model.IncomeConnection{}
	err = q.cachedList(ctx, "test_incomes_miss", struct{}{}, &result, func() (interface{}, error) {
		callCount++
		return &model.IncomeConnection{
			Edges:      []*model.IncomeEdge{{Node: validTestIncome("wrong", "should not be called")}},
			TotalCount: 999,
		}, nil
	})
	require.NoError(t, err)
	assert.Equal(t, 1, callCount, "fetchFn should NOT be called — value should be cached")
	assert.Equal(t, 5, result.TotalCount)
}

// CC-15: Cache TTL expiry — after the cache entry is deleted (simulating expiry),
// cachedSingle should call the fallback again (cache miss).
func TestCachedSingle_CacheTTLExpiry(t *testing.T) {
	cacheStore := newTestCache(t)
	r := &Resolver{Cache: cacheStore}
	q := &queryResolver{r}
	ctx := context.Background()

	callCount := 0

	// First call — cache MISS → stores result
	var result model.Income
	err := q.cachedSingle(ctx, "test_ttl", "id-1", &result, func() (interface{}, error) {
		callCount++
		return validTestIncome("id-1", "original"), nil
	})
	require.NoError(t, err)
	assert.Equal(t, 1, callCount)
	assert.Equal(t, "original", result.Description)

	// Simulate expiry by deleting the cache entry
	err = cacheStore.Delete(ctx, cache.CacheKey("test_ttl", "id-1"))
	require.NoError(t, err)

	// Now the cache should miss again and call fetchFn
	result = model.Income{}
	err = q.cachedSingle(ctx, "test_ttl", "id-1", &result, func() (interface{}, error) {
		callCount++
		return validTestIncome("id-1", "refreshed"), nil
	})
	require.NoError(t, err)
	assert.Equal(t, 2, callCount, "fetchFn should be called again after TTL expiry")
	assert.Equal(t, "refreshed", result.Description)
}

// CC-17: Non-existent key — fetchFn returning error should propagate.
func TestCachedSingle_NotFoundWithCache(t *testing.T) {
	cacheStore := newTestCache(t)
	r := &Resolver{Cache: cacheStore}
	q := &queryResolver{r}
	ctx := context.Background()

	var result model.Income
	err := q.cachedSingle(ctx, "test_notfound", "missing", &result, func() (interface{}, error) {
		return nil, fmt.Errorf("income not found")
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "income not found")
}

// CC-17: Empty list — cachedList with an empty result should succeed and cache
// the empty result so subsequent calls hit the cache.
func TestCachedList_EmptyList(t *testing.T) {
	cacheStore := newTestCache(t)
	r := &Resolver{Cache: cacheStore}
	q := &queryResolver{r}
	ctx := context.Background()

	callCount := 0

	// First call — cache MISS → stores empty list
	var result model.IncomeConnection
	err := q.cachedList(ctx, "test_empty", struct{}{}, &result, func() (interface{}, error) {
		callCount++
		return &model.IncomeConnection{
			Edges:      []*model.IncomeEdge{},
			TotalCount: 0,
			PageInfo:   &model.PageInfo{HasNextPage: false},
		}, nil
	})
	require.NoError(t, err)
	assert.Equal(t, 1, callCount)
	assert.Equal(t, 0, result.TotalCount)
	assert.Len(t, result.Edges, 0)

	// Second call — cache HIT → fetchFn NOT called, empty list returned from cache
	result = model.IncomeConnection{}
	err = q.cachedList(ctx, "test_empty", struct{}{}, &result, func() (interface{}, error) {
		callCount++
		return &model.IncomeConnection{
			Edges:      []*model.IncomeEdge{{Node: validTestIncome("wrong", "should not be called")}},
			TotalCount: 999,
		}, nil
	})
	require.NoError(t, err)
	assert.Equal(t, 1, callCount, "fetchFn should NOT be called — empty list should be cached")
	assert.Equal(t, 0, result.TotalCount)
	assert.Len(t, result.Edges, 0)
}
