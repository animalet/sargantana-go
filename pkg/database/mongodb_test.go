//go:build unit

package database

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("MongoDBConfig", func() {
	Context("Validation", func() {
		It("should validate correct configuration", func() {
			cfg := MongoDBConfig{
				URI:      "mongodb://localhost:27017",
				Database: "testdb",
			}
			Expect(cfg.Validate()).To(Succeed())
		})

		It("should fail if URI is missing", func() {
			cfg := MongoDBConfig{
				Database: "testdb",
			}
			Expect(cfg.Validate()).To(HaveOccurred())
		})

		It("should fail if database is missing", func() {
			cfg := MongoDBConfig{
				URI: "mongodb://localhost:27017",
			}
			Expect(cfg.Validate()).To(HaveOccurred())
		})

		It("should fail if connect_timeout is negative", func() {
			cfg := MongoDBConfig{
				URI:            "mongodb://localhost:27017",
				Database:       "testdb",
				ConnectTimeout: -1,
			}
			Expect(cfg.Validate()).To(HaveOccurred())
		})

		It("should fail if min_pool_size > max_pool_size", func() {
			cfg := MongoDBConfig{
				URI:         "mongodb://localhost:27017",
				Database:    "testdb",
				MaxPoolSize: 10,
				MinPoolSize: 20,
			}
			Expect(cfg.Validate()).To(HaveOccurred())
		})

		Context("TLS Validation", func() {
			var tempDir string

			BeforeEach(func() {
				var err error
				tempDir, err = os.MkdirTemp("", "mongo-tls")
				Expect(err).NotTo(HaveOccurred())
			})

			AfterEach(func() {
				os.RemoveAll(tempDir)
			})

			It("should fail if cert_file is set without key_file", func() {
				cfg := MongoDBConfig{
					URI:      "mongodb://localhost:27017",
					Database: "testdb",
					TLS: &MongoDBTLSConfig{
						CertFile: "cert.pem",
					},
				}
				Expect(cfg.Validate()).To(HaveOccurred())
			})

			It("should fail if key_file is set without cert_file", func() {
				cfg := MongoDBConfig{
					URI:      "mongodb://localhost:27017",
					Database: "testdb",
					TLS: &MongoDBTLSConfig{
						KeyFile: "key.pem",
					},
				}
				Expect(cfg.Validate()).To(HaveOccurred())
			})

			It("should fail if cert_file does not exist", func() {
				cfg := MongoDBConfig{
					URI:      "mongodb://localhost:27017",
					Database: "testdb",
					TLS: &MongoDBTLSConfig{
						CertFile: "/non/existent/cert.pem",
						KeyFile:  "/non/existent/key.pem",
					},
				}
				Expect(cfg.Validate()).To(HaveOccurred())
			})

			It("should validate correct TLS config with existing files", func() {
				certPath := filepath.Join(tempDir, "cert.pem")
				keyPath := filepath.Join(tempDir, "key.pem")
				caPath := filepath.Join(tempDir, "ca.pem")

				err := os.WriteFile(certPath, []byte("cert"), 0644)
				Expect(err).NotTo(HaveOccurred())
				err = os.WriteFile(keyPath, []byte("key"), 0644)
				Expect(err).NotTo(HaveOccurred())
				err = os.WriteFile(caPath, []byte("ca"), 0644)
				Expect(err).NotTo(HaveOccurred())

				cfg := MongoDBConfig{
					URI:      "mongodb://localhost:27017",
					Database: "testdb",
					TLS: &MongoDBTLSConfig{
						CertFile: certPath,
						KeyFile:  keyPath,
						CAFile:   caPath,
					},
				}
				Expect(cfg.Validate()).To(Succeed())
			})
		})
	})
})
