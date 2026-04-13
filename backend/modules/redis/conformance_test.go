//go:build integration

package redis_test

import (
	"context"
	"testing"

	goredis "github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"

	"github.com/orlandoburli/feature-bacon/internal/conformance"
	redisstore "github.com/orlandoburli/feature-bacon/modules/redis/store"
)

func TestPersistenceConformance(t *testing.T) {
	ctx := context.Background()

	container, err := tcredis.Run(ctx, "redis:7-alpine")
	testcontainers.CleanupContainer(t, container)
	if err != nil {
		t.Fatalf("start redis container: %v", err)
	}

	connStr, err := container.ConnectionString(ctx)
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}

	opts, err := goredis.ParseURL(connStr)
	if err != nil {
		t.Fatalf("parse redis URL: %v", err)
	}

	client := goredis.NewClient(opts)
	defer client.Close()

	st := redisstore.New(client)
	conformance.RunPersistenceSuite(t, st)
}
