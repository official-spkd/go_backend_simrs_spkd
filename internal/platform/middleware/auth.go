package middleware

import (
	"net/http"
	"slices"
	"strings"

	"github.com/gin-gonic/gin"

	"gin-quickstart/internal/platform/utils"
)

const claimsKey = "auth.claims"

// Required rejects requests without a valid Bearer token.
func Required(tokens *utils.TokenManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetHeader("Authorization") == "" {
			utils.AbortWithError(c, http.StatusUnauthorized, "missing authorization header")
			return
		}
		authenticate(c, tokens)
	}
}

// Optional lets anonymous requests through, but rejects a token that is present and invalid.
func Optional(tokens *utils.TokenManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetHeader("Authorization") == "" {
			c.Next()
			return
		}
		authenticate(c, tokens)
	}
}

// RequireRoles allows only the given role names. It must run after Required.
func RequireRoles(names ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, ok := ClaimsFrom(c)
		if !ok {
			utils.AbortWithError(c, http.StatusUnauthorized, "unauthenticated")
			return
		}
		if !slices.Contains(names, claims.Role) {
			utils.AbortWithError(c, http.StatusForbidden, "insufficient permissions")
			return
		}
		c.Next()
	}
}

// ClaimsFrom returns the claims of the authenticated user, if any.
func ClaimsFrom(c *gin.Context) (*utils.TokenClaims, bool) {
	v, ok := c.Get(claimsKey)
	if !ok {
		return nil, false
	}
	claims, ok := v.(*utils.TokenClaims)
	return claims, ok
}

func authenticate(c *gin.Context, tokens *utils.TokenManager) {
	raw, found := strings.CutPrefix(c.GetHeader("Authorization"), "Bearer ")
	if !found || raw == "" {
		utils.AbortWithError(c, http.StatusUnauthorized, "malformed authorization header")
		return
	}
	claims, err := tokens.Parse(raw)
	if err != nil {
		utils.AbortWithError(c, http.StatusUnauthorized, err.Error())
		return
	}
	c.Set(claimsKey, claims)
	c.Next()
}
