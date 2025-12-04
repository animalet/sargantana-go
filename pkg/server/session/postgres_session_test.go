//go:build unit

package session

import (
	"github.com/jackc/pgx/v5/pgxpool"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Postgres Session Store", func() {
	Context("NewPostgresSessionStore", func() {
		It("should return error if pool is nil", func() {
			_, err := NewPostgresSessionStore(true, []byte("secret"), nil, "table")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("PostgreSQL pool cannot be nil"))
		})

		It("should return error if secret is empty", func() {
			pool := &pgxpool.Pool{}
			_, err := NewPostgresSessionStore(true, nil, pool, "table")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("session secret cannot be empty"))
		})
	})
})
