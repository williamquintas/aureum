package clients

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

func TestNewTransactionServiceClient(t *testing.T) {
	conn, err := grpc.Dial("localhost:9999", grpc.WithInsecure())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	client := NewTransactionServiceClient(conn)
	assert.NotNil(t, client)
	assert.NotNil(t, client.raw)
	assert.NotNil(t, client.cb)
}

func TestTransactionServiceClient_Timeout(t *testing.T) {
	conn, err := grpc.Dial("localhost:9999", grpc.WithInsecure())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	client := NewTransactionServiceClient(conn)
	assert.Equal(t, "5s", client.Timeout().String())
}

func TestTransactionServiceClient_CircuitBreakerConfig(t *testing.T) {
	conn, err := grpc.Dial("localhost:9999", grpc.WithInsecure())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	client := NewTransactionServiceClient(conn)
	// Circuit breaker should be in closed state initially
	assert.Equal(t, "closed", client.cb.State().String())
}
