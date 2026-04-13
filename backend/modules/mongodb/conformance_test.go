//go:build integration

package mongodb_test

import (
	"context"
	"testing"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/testcontainers/testcontainers-go"
	tcmongo "github.com/testcontainers/testcontainers-go/modules/mongodb"

	"github.com/orlandoburli/feature-bacon/internal/conformance"
	mongostore "github.com/orlandoburli/feature-bacon/modules/mongodb/store"
)

func TestPersistenceConformance(t *testing.T) {
	ctx := context.Background()

	container, err := tcmongo.Run(ctx, "mongo:7")
	testcontainers.CleanupContainer(t, container)
	if err != nil {
		t.Fatalf("start mongo container: %v", err)
	}

	connStr, err := container.ConnectionString(ctx)
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}

	client, err := mongo.Connect(options.Client().ApplyURI(connStr))
	if err != nil {
		t.Fatalf("connect to mongo: %v", err)
	}
	defer func() { _ = client.Disconnect(ctx) }()

	db := client.Database("test_conformance")
	if err := mongostore.EnsureIndexes(ctx, db); err != nil {
		t.Fatalf("ensure indexes: %v", err)
	}

	st := mongostore.New(db)
	conformance.RunPersistenceSuite(t, st)
}
