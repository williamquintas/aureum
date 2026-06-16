package cache

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCacheKey(t *testing.T) {
	tests := []struct {
		name     string
		entity   string
		id       string
		expected string
	}{
		{
			name:     "income entity",
			entity:   "income",
			id:       "123",
			expected: "graphql-bff:income:123",
		},
		{
			name:     "user entity",
			entity:   "user",
			id:       "user-456",
			expected: "graphql-bff:user:user-456",
		},
		{
			name:     "fixed expense entity",
			entity:   "fixed_expense",
			id:       "789",
			expected: "graphql-bff:fixed_expense:789",
		},
		{
			name:     "empty id",
			entity:   "test",
			id:       "",
			expected: "graphql-bff:test:",
		},
		{
			name:     "empty entity",
			entity:   "",
			id:       "123",
			expected: "graphql-bff::123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CacheKey(tt.entity, tt.id)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCacheKeyList(t *testing.T) {
	tests := []struct {
		name     string
		entity   string
		args     []interface{}
		expected string
	}{
		{
			name:     "with int arg",
			entity:   "incomes",
			args:     []interface{}{20, 0},
			expected: "graphql-bff:incomes:list:[20 0]",
		},
		{
			name:     "with string arg",
			entity:   "variable_expenses",
			args:     []interface{}{"food", 10},
			expected: "graphql-bff:variable_expenses:list:[food 10]",
		},
		{
			name:     "no args",
			entity:   "test",
			args:     []interface{}{},
			expected: "graphql-bff:test:list:[]",
		},
		{
			name:     "with nil arg",
			entity:   "incomes",
			args:     []interface{}{nil, "abc"},
			expected: "graphql-bff:incomes:list:[<nil> abc]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CacheKeyList(tt.entity, tt.args...)
			assert.Equal(t, tt.expected, result)
		})
	}
}
