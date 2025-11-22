//go:build integration

package database_test

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/animalet/sargantana-go/pkg/database"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/readpref"
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

	It("should connect with TLS and user/pass", func() {
		cwd, _ := os.Getwd()
		projectRoot := filepath.Dir(filepath.Dir(cwd))
		certsDir := filepath.Join(projectRoot, "certs")

		cfg := database.MongoDBConfig{
			URI:        "mongodb://localhost:27018",
			Database:   "sessions_test",
			Username:   "testuser",
			Password:   "testpass",
			AuthSource: "admin",
			TLS: &database.MongoDBTLSConfig{
				CAFile:             filepath.Join(certsDir, "ca.crt"),
				CertFile:           filepath.Join(certsDir, "client.crt"),
				KeyFile:            filepath.Join(certsDir, "client.key"),
				InsecureSkipVerify: true, // Hostname matching might fail with localhost vs mongodb-tls
			},
		}

		client, err := cfg.CreateClient()
		Expect(err).NotTo(HaveOccurred())
		defer client.Disconnect(context.Background())

		err = client.Ping(context.Background(), readpref.Primary())
		Expect(err).NotTo(HaveOccurred())
	})
})
