package middleware

import (
	"github.com/gin-gonic/gin"
)

// SecurityHeaders adds common security headers to the response.
func SecurityHeaders(cspConfig string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Protects against MIME sniffing vulnerabilities
		c.Header("X-Content-Type-Options", "nosniff")

		// Protects against Clickjacking attacks
		c.Header("X-Frame-Options", "DENY")

		// Enables the Cross-Site Scripting (XSS) filter built into most modern web browsers
		c.Header("X-XSS-Protection", "1; mode=block")

		// Strict-Transport-Security (HSTS) enforces secure (HTTP over SSL/TLS) connections to the server
		// This is set to 1 year (31536000 seconds) including subdomains
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

		// Content-Security-Policy (CSP) is an added layer of security that helps to detect and mitigate certain types of attacks,
		// including Cross-Site Scripting (XSS) and data injection attacks.
		if cspConfig != "" {
			c.Header("Content-Security-Policy", cspConfig)
		} else {
			// This is a starting point and might need adjustment based on application needs.
			c.Header("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self'; object-src 'none'; frame-ancestors 'none'; base-uri 'self'; form-action 'self';")
		}

		// Referrer-Policy controls how much referrer information (sent via the Referer header) should be included with requests
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		// Permissions-Policy allows a site to allow or block the use of browser features in its own frame or in iframes that it embeds.
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=(), payment=()")

		c.Next()
	}
}
