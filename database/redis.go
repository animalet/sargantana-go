package database

import (
	"time"

	"github.com/gomodule/redigo/redis"
)

func NewRedisPool(address string) *redis.Pool {
	return &redis.Pool{
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			if time.Since(t) < time.Minute {
				return nil
			}
			_, err := c.Do("PING")
			return err
		},
		MaxIdle:     10,
		IdleTimeout: 240 * time.Second,
		Dial:        func() (redis.Conn, error) { return redis.Dial("tcp", address) },
	}
}
