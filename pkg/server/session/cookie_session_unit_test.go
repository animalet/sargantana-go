//go:build unit

package session

import (
	"github.com/gin-contrib/sessions/cookie"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Cookie Session Store", func() {
	Context("NewCookieStore", func() {
		It("should create a new cookie store with correct options", func() {
			secret := []byte("secret-key")
			store := NewCookieStore(true, secret)
			Expect(store).NotTo(BeNil())

			// Type assertion to check if it's a cookie store
			cookieStore, ok := store.(cookie.Store)
			Expect(ok).To(BeTrue())
			Expect(cookieStore).NotTo(BeNil())
		})

		It("should create store in non-secure mode", func() {
			secret := []byte("secret-key")
			store := NewCookieStore(false, secret)
			Expect(store).NotTo(BeNil())
		})
	})
})
