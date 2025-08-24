package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"sitewatch/internal/models"
)

// Service handles authentication operations
type Service struct {
	config *models.AuthConfig
}

// NewService creates a new authentication service
func NewService(config *models.AuthConfig) *Service {
	return &Service{
		config: config,
	}
}

// IsEnabled returns whether authentication is enabled
func (s *Service) IsEnabled() bool {
	return s.config != nil && s.config.Enabled
}

// ValidateUISecret validates UI session secret
func (s *Service) ValidateUISecret(secret string) bool {
	if !s.IsEnabled() {
		return true // Auth disabled, allow all
	}
	
	return s.config.UI.Secret != "" && s.config.UI.Secret == secret
}

// ValidateAPIToken validates API token and returns token info
func (s *Service) ValidateAPIToken(tokenString string) (*models.APIToken, error) {
	if !s.IsEnabled() {
		return &models.APIToken{
			Token:       "disabled",
			Name:        "Authentication Disabled",
			Permissions: []string{"metrics", "read", "admin"},
		}, nil
	}

	for _, token := range s.config.API.Tokens {
		if token.Token == tokenString {
			if token.IsExpired() {
				return nil, fmt.Errorf("token expired")
			}
			return &token, nil
		}
	}

	return nil, fmt.Errorf("invalid token")
}

// HasPermission checks if token has required permission
func (s *Service) HasPermission(token *models.APIToken, permission models.TokenPermission) bool {
	if !s.IsEnabled() {
		return true
	}

	return token != nil && token.HasPermission(permission)
}

// GetUISessionName returns the UI session cookie name
func (s *Service) GetUISessionName() string {
	if s.config == nil || s.config.UI.SessionName == "" {
		return "sitewatch_session"
	}
	return s.config.UI.SessionName
}

// GetUISessionExpiry returns the UI session expiry duration
func (s *Service) GetUISessionExpiry() time.Duration {
	if s.config == nil || s.config.UI.ExpiresHours == 0 {
		return 24 * time.Hour // Default 24 hours
	}
	return time.Duration(s.config.UI.ExpiresHours) * time.Hour
}

// GenerateToken generates a new secure token
func GenerateToken(prefix string) (string, error) {
	bytes := make([]byte, 16) // 128 bit
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	
	token := prefix + "_" + hex.EncodeToString(bytes)
	return token, nil
}

// GenerateUISecret generates a secure UI secret
func GenerateUISecret() (string, error) {
	bytes := make([]byte, 32) // 256 bit
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	
	return hex.EncodeToString(bytes), nil
}