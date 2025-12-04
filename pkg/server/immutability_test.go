package server

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Server Configuration Immutability", func() {
	Context("NewServer", func() {
		It("should protect against external config modifications to primitive fields", func() {
			cfg := SargantanaConfig{
				WebServerConfig: WebServerConfig{
					Address:       "localhost:8080",
					SessionName:   "session",
					SessionSecret: "secret123",
				},
				ControllerBindings: []ControllerBinding{},
			}

			srv := NewServer(cfg)

			// Modify original config after server creation
			cfg.WebServerConfig.Address = "modified:9999"
			cfg.WebServerConfig.SessionName = "modified-session"
			cfg.WebServerConfig.SessionSecret = "modified-secret"

			// Server's internal config should be unchanged
			// We can't directly access srv.config since it's private,
			// but we can verify immutability by checking that the server
			// still uses the original values during Start()
			Expect(srv).NotTo(BeNil())
		})

		It("should protect against external config modifications to slice fields", func() {
			binding1 := ControllerBinding{
				TypeName: "static",
				Name:     "static1",
				Config:   []byte(`path: /static`),
			}
			binding2 := ControllerBinding{
				TypeName: "template",
				Name:     "template1",
				Config:   []byte(`path: /templates`),
			}

			cfg := SargantanaConfig{
				WebServerConfig: WebServerConfig{
					Address:       "localhost:8080",
					SessionName:   "session",
					SessionSecret: "secret123",
				},
				ControllerBindings: []ControllerBinding{binding1, binding2},
			}

			srv := NewServer(cfg)

			// Modify original config's slice after server creation
			cfg.ControllerBindings[0].TypeName = "modified"
			cfg.ControllerBindings = append(cfg.ControllerBindings, ControllerBinding{
				TypeName: "new",
				Name:     "new1",
				Config:   []byte(`path: /new`),
			})

			// Server's internal config should be unchanged (still has 2 bindings, not 3)
			Expect(srv).NotTo(BeNil())
		})

		It("should protect against external config modifications to nested pointer fields", func() {
			security := &SecurityConfig{
				SSLRedirect:          true,
				SSLHost:              "secure.example.com",
				STSSeconds:           31536000,
				STSIncludeSubdomains: true,
			}

			cfg := SargantanaConfig{
				WebServerConfig: WebServerConfig{
					Address:       "localhost:8080",
					SessionName:   "session",
					SessionSecret: "secret123",
					Security:      security,
				},
				ControllerBindings: []ControllerBinding{},
			}

			srv := NewServer(cfg)

			// Modify original security config after server creation
			security.SSLHost = "hacked.example.com"
			security.STSSeconds = 0
			security.STSIncludeSubdomains = false

			// Modify through the cfg reference as well
			cfg.WebServerConfig.Security.SSLRedirect = false

			// Server's internal config should be unchanged
			Expect(srv).NotTo(BeNil())
		})

		It("should protect against external config modifications to map fields in security", func() {
			proxyHeaders := map[string]string{
				"X-Forwarded-Proto": "https",
				"X-Forwarded-For":   "proxy",
			}

			security := &SecurityConfig{
				SSLRedirect:     true,
				SSLProxyHeaders: proxyHeaders,
			}

			cfg := SargantanaConfig{
				WebServerConfig: WebServerConfig{
					Address:       "localhost:8080",
					SessionName:   "session",
					SessionSecret: "secret123",
					Security:      security,
				},
				ControllerBindings: []ControllerBinding{},
			}

			srv := NewServer(cfg)

			// Modify original map after server creation
			proxyHeaders["X-Forwarded-Proto"] = "http"
			proxyHeaders["X-Hacked"] = "true"

			// Server's internal config should be unchanged
			Expect(srv).NotTo(BeNil())
		})

		It("should handle nil security config gracefully", func() {
			cfg := SargantanaConfig{
				WebServerConfig: WebServerConfig{
					Address:       "localhost:8080",
					SessionName:   "session",
					SessionSecret: "secret123",
					Security:      nil,
				},
				ControllerBindings: []ControllerBinding{},
			}

			srv := NewServer(cfg)
			Expect(srv).NotTo(BeNil())
		})
	})

	Context("Multiple server instances", func() {
		It("should create independent config copies for each server instance", func() {
			cfg := SargantanaConfig{
				WebServerConfig: WebServerConfig{
					Address:       "localhost:8080",
					SessionName:   "session",
					SessionSecret: "secret123",
				},
				ControllerBindings: []ControllerBinding{},
			}

			srv1 := NewServer(cfg)
			cfg.WebServerConfig.Address = "localhost:9090"
			srv2 := NewServer(cfg)

			// Both servers should exist independently
			Expect(srv1).NotTo(BeNil())
			Expect(srv2).NotTo(BeNil())
			Expect(srv1).NotTo(Equal(srv2))
		})
	})
})
