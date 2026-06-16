package featureflag

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewClient_InvalidURL(t *testing.T) {
	client, err := NewClient("", "graphql-bff", "")
	assert.Error(t, err)
	assert.Nil(t, client)
}

func TestNewClient_InvalidAppName(t *testing.T) {
	client, err := NewClient("http://localhost:4242", "", "")
	assert.Error(t, err)
	assert.Nil(t, client)
}

func TestNewClient_TokenWithoutURL(t *testing.T) {
	client, err := NewClient("", "graphql-bff", "test-token")
	assert.Error(t, err)
	assert.Nil(t, client)
}
