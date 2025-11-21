//go:build integration

package database_test

import (
	"context"
	"time"

	"github.com/animalet/sargantana-go/pkg/database"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/bson"
)

var _ = Describe("MongoDB Integration", func() {
	It("should connect to mongodb and perform operations", func() {
		cfg := database.MongoDBConfig{
			URI:            "mongodb://localhost:27017",
			Database:       "sessions_test",
			Username:       "testuser",
			Password:       "testpass",
			AuthSource:     "admin",
			ConnectTimeout: 10 * time.Second,
		}

		client, err := cfg.CreateClient()
		Expect(err).NotTo(HaveOccurred())
		defer client.Disconnect(context.Background())

		db := client.Database("sessions_test")
		collection := db.Collection("test_collection")

		_, err = collection.InsertOne(context.Background(), bson.M{"key": "value"})
		Expect(err).NotTo(HaveOccurred())

		var result bson.M
		err = collection.FindOne(context.Background(), bson.M{"key": "value"}).Decode(&result)
		Expect(err).NotTo(HaveOccurred())
		Expect(result["key"]).To(Equal("value"))
	})
})
