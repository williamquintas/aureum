package testutils

import (
	"context"
	"fmt"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const testDBName = "test"

type PostgreSQLContainer struct {
	container testcontainers.Container
	URI       string
}

type RedisContainer struct {
	container testcontainers.Container
	URI       string
}

type KeycloakContainer struct {
	container testcontainers.Container
	URI       string
}

type RedpandaContainer struct {
	container testcontainers.Container
	URI       string
}

func (c *PostgreSQLContainer) Close() error {
	return c.container.Terminate(context.Background())
}

func (c *RedisContainer) Close() error {
	return c.container.Terminate(context.Background())
}

func (c *KeycloakContainer) Close() error {
	return c.container.Terminate(context.Background())
}

func (c *RedpandaContainer) Close() error {
	return c.container.Terminate(context.Background())
}

func NewPostgreSQLContainer(ctx context.Context) (*PostgreSQLContainer, error) {
	req := testcontainers.ContainerRequest{
		Image: "postgres:16-alpine",
		Env: map[string]string{
			"POSTGRES_USER":     testDBName,
			"POSTGRES_PASSWORD": testDBName,
			"POSTGRES_DB":       testDBName,
		},
		ExposedPorts: []string{"5432/tcp"},
		WaitingFor: wait.ForLog("database system is ready to accept connections").
			WithOccurrence(2).
			WithStartupTimeout(60 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, err
	}

	host, err := container.Host(ctx)
	if err != nil {
		return nil, err
	}

	port, err := container.MappedPort(ctx, "5432")
	if err != nil {
		return nil, err
	}

	uri := fmt.Sprintf("postgres://test:test@%s:%s/test?sslmode=disable", host, port.Port())

	return &PostgreSQLContainer{container: container, URI: uri}, nil
}

func NewRedisContainer(ctx context.Context) (*RedisContainer, error) {
	req := testcontainers.ContainerRequest{
		Image:        "redis:7-alpine",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForLog("Ready to accept connections tcp").WithStartupTimeout(60 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, err
	}

	host, err := container.Host(ctx)
	if err != nil {
		return nil, err
	}

	port, err := container.MappedPort(ctx, "6379")
	if err != nil {
		return nil, err
	}

	uri := fmt.Sprintf("redis://%s:%s", host, port.Port())

	return &RedisContainer{container: container, URI: uri}, nil
}

func NewKeycloakContainer(ctx context.Context) (*KeycloakContainer, error) {
	req := testcontainers.ContainerRequest{
		Image: "quay.io/keycloak/keycloak:25.0",
		Env: map[string]string{
			"KC_BOOTSTRAP_ADMIN_USERNAME": "admin",
			"KC_BOOTSTRAP_ADMIN_PASSWORD": "admin",
		},
		ExposedPorts: []string{"8080/tcp"},
		WaitingFor:   wait.ForHTTP("/").WithPort("8080/tcp").WithStartupTimeout(120 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, err
	}

	host, err := container.Host(ctx)
	if err != nil {
		return nil, err
	}

	port, err := container.MappedPort(ctx, "8080")
	if err != nil {
		return nil, err
	}

	uri := fmt.Sprintf("http://%s:%s", host, port.Port())

	return &KeycloakContainer{container: container, URI: uri}, nil
}

func NewRedpandaContainer(ctx context.Context) (*RedpandaContainer, error) {
	req := testcontainers.ContainerRequest{
		Image:        "docker.redpanda.com/redpandadata/redpanda:v24.1",
		ExposedPorts: []string{"9092/tcp"},
		Cmd: []string{
			"redpanda", "start",
			"--smp", "1",
			"--memory", "512M",
			"--overprovisioned",
			"--kafka-addr", "0.0.0.0:9092",
			"--advertise-kafka-addr", "localhost:9092",
		},
		WaitingFor: wait.ForLog("Successfully started Redpanda").WithStartupTimeout(120 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, err
	}

	host, err := container.Host(ctx)
	if err != nil {
		return nil, err
	}

	port, err := container.MappedPort(ctx, "9092")
	if err != nil {
		return nil, err
	}

	uri := fmt.Sprintf("%s:%s", host, port.Port())

	return &RedpandaContainer{container: container, URI: uri}, nil
}
