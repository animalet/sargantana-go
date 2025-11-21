//go:build integration

package database_test

import (
	"time"

	"github.com/animalet/sargantana-go/pkg/database"
	"github.com/gomodule/redigo/redis"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Redis Integration", func() {
	It("should connect to redis and perform operations", func() {
		cfg := database.RedisConfig{
			Address:     "localhost:6379",
			Username:    "redisuser",
			Password:    "redispass",
			MaxIdle:     3,
			IdleTimeout: 240 * time.Second,
		}

		pool, err := cfg.CreateClient()
		Expect(err).NotTo(HaveOccurred())
		defer pool.Close()

		conn := pool.Get()
		defer conn.Close()

		_, err = conn.Do("SET", "test-key", "test-value")
		Expect(err).NotTo(HaveOccurred())

		val, err := redis.String(conn.Do("GET", "test-key"))
		Expect(err).NotTo(HaveOccurred())
		Expect(val).To(Equal("test-value"))
	})
})
