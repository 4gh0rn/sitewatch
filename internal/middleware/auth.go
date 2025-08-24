package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"sitewatch/internal/models"
	"sitewatch/internal/services/auth"
)

// AuthContext stores authentication info in request context
type AuthContext struct {
	IsAuthenticated bool
	Token          *models.APIToken
	AuthType       string // "ui" or "api"
}

// UIAuthMiddleware validates UI session cookies
func UIAuthMiddleware(authService *auth.Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Skip if auth is disabled
		if !authService.IsEnabled() {
			c.Locals("auth", &AuthContext{
				IsAuthenticated: true,
				AuthType:       "disabled",
			})
			return c.Next()
		}

		// Get session cookie
		sessionName := authService.GetUISessionName()
		sessionSecret := c.Cookies(sessionName)

		if sessionSecret == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "UI session required",
				"code":  "NO_SESSION",
			})
		}

		// Validate UI secret
		if !authService.ValidateUISecret(sessionSecret) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid UI session",
				"code":  "INVALID_SESSION",
			})
		}

		// Store auth context
		c.Locals("auth", &AuthContext{
			IsAuthenticated: true,
			AuthType:       "ui",
		})

		return c.Next()
	}
}

// APIAuthMiddleware validates API tokens from Authorization header
func APIAuthMiddleware(authService *auth.Service, requiredPermission models.TokenPermission) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Skip if auth is disabled
		if !authService.IsEnabled() {
			c.Locals("auth", &AuthContext{
				IsAuthenticated: true,
				Token: &models.APIToken{
					Token:       "disabled",
					Name:        "Authentication Disabled",
					Permissions: []string{"metrics", "read", "admin"},
				},
				AuthType: "disabled",
			})
			return c.Next()
		}

		// Get Authorization header
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Authorization header required",
				"code":  "NO_TOKEN",
			})
		}

		// Extract Bearer token
		tokenString := ""
		if strings.HasPrefix(authHeader, "Bearer ") {
			tokenString = strings.TrimPrefix(authHeader, "Bearer ")
		} else {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Bearer token required",
				"code":  "INVALID_TOKEN_FORMAT",
			})
		}

		// Validate token
		token, err := authService.ValidateAPIToken(tokenString)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid token: " + err.Error(),
				"code":  "INVALID_TOKEN",
			})
		}

		// Check permissions
		if !authService.HasPermission(token, requiredPermission) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error":      "Insufficient permissions",
				"code":       "INSUFFICIENT_PERMISSIONS", 
				"required":   string(requiredPermission),
				"available":  token.Permissions,
			})
		}

		// Store auth context
		c.Locals("auth", &AuthContext{
			IsAuthenticated: true,
			Token:          token,
			AuthType:       "api",
		})

		return c.Next()
	}
}

// GetAuthContext retrieves authentication context from request
func GetAuthContext(c *fiber.Ctx) *AuthContext {
	if auth, ok := c.Locals("auth").(*AuthContext); ok {
		return auth
	}
	return &AuthContext{
		IsAuthenticated: false,
		AuthType:       "none",
	}
}