//go:build unit

package session

import (
	"github.com/gomodule/redigo/redis"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// MockRedisConn implements redis.Conn for testing
type MockRedisConn struct{}

func (m *MockRedisConn) Close() error { return nil }
func (m *MockRedisConn) Err() error   { return nil }
func (m *MockRedisConn) Do(commandName string, args ...interface{}) (interface{}, error) {
	return "OK", nil
}
func (m *MockRedisConn) Send(commandName string, args ...interface{}) error { return nil }
func (m *MockRedisConn) Flush() error                                       { return nil }
func (m *MockRedisConn) Receive() (interface{}, error)                      { return nil, nil }

var _ = Describe("Redis Session Store", func() {
	Context("NewRedisSessionStore", func() {
		It("should create a new redis store with correct options", func() {
			secret := []byte("secret-key")
			pool := &redis.Pool{
				Dial: func() (redis.Conn, error) {
					return &MockRedisConn{}, nil
				},
			}

			store, err := NewRedisSessionStore(true, secret, pool)
			Expect(err).NotTo(HaveOccurred())
			Expect(store).NotTo(BeNil())
		})

		It("should return error if pool is nil", func() {
			_, err := NewRedisSessionStore(true, []byte("secret"), nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Redis pool cannot be nil"))
		})

		It("should create store in non-secure mode", func() {
			secret := []byte("secret-key-123")
			pool := &redis.Pool{
				Dial: func() (redis.Conn, error) {
					return &MockRedisConn{}, nil
				},
			}

			store, err := NewRedisSessionStore(false, secret, pool)
			Expect(err).NotTo(HaveOccurred())
			Expect(store).NotTo(BeNil())
		})
	})
})
