package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	// The API returns JSON only, so nothing may be loaded or framed.
	apiCSP = "default-src 'none'; frame-ancestors 'none'; base-uri 'none'; form-action 'none'"

	// Swagger UI needs its own scripts and inline styles/scripts to render.
	swaggerCSP = "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; " +
		"img-src 'self' data:; font-src 'self' data:; frame-ancestors 'none'; base-uri 'self'; form-action 'self'"

	hstsValue = "max-age=63072000; includeSubDomains"
)

func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.Writer.Header()

		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=()")
		h.Set("Cross-Origin-Opener-Policy", "same-origin")
		h.Set("Cross-Origin-Resource-Policy", "same-origin")
		h.Set("X-Permitted-Cross-Domain-Policies", "none")

		if isHTTPS(c) {
			h.Set("Strict-Transport-Security", hstsValue)
		}

		if strings.HasPrefix(c.Request.URL.Path, "/swagger") {
			h.Set("Content-Security-Policy", swaggerCSP)
		} else {
			h.Set("Content-Security-Policy", apiCSP)
			// API responses carry personal and auth data and must not be cached.
			h.Set("Cache-Control", "no-store")
		}

		c.Next()
	}
}

func isHTTPS(c *gin.Context) bool {
	return c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https"
}
