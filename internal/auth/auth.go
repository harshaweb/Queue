package auth

import (
	"context"
	"crypto/rsa"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// Authenticator provides authentication and authorization
type Authenticator interface {
	ValidateToken(ctx context.Context, token string) (*User, error)
	HasAdminRole(user interface{}) bool
	GenerateToken(user *User) (string, error)
}

// User represents an authenticated user
type User struct {
	ID       string   `json:"id"`
	Username string   `json:"username"`
	Email    string   `json:"email"`
	Roles    []string `json:"roles"`
}

// JWTAuthenticator implements JWT-based authentication
type JWTAuthenticator struct {
	publicKey  *rsa.PublicKey
	privateKey *rsa.PrivateKey
	issuer     string
	audience   string
}

// NewJWTAuthenticator creates a new JWT authenticator
func NewJWTAuthenticator(publicKey *rsa.PublicKey, privateKey *rsa.PrivateKey, issuer, audience string) *JWTAuthenticator {
	return &JWTAuthenticator{
		publicKey:  publicKey,
		privateKey: privateKey,
		issuer:     issuer,
		audience:   audience,
	}
}

// Claims represents JWT claims
type Claims struct {
	UserID   string   `json:"user_id"`
	Username string   `json:"username"`
	Email    string   `json:"email"`
	Roles    []string `json:"roles"`
	jwt.RegisteredClaims
}

// ValidateToken validates a JWT token and returns the user
func (j *JWTAuthenticator) ValidateToken(ctx context.Context, tokenString string) (*User, error) {
	// Remove "Bearer " prefix if present
	tokenString = strings.TrimPrefix(tokenString, "Bearer ")

	// Parse token
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return j.publicKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	// Validate claims
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	// Check issuer and audience
	if claims.Issuer != j.issuer {
		return nil, fmt.Errorf("invalid issuer")
	}

	if claims.Audience[0] != j.audience {
		return nil, fmt.Errorf("invalid audience")
	}

	// Check expiration
	if time.Now().After(claims.ExpiresAt.Time) {
		return nil, fmt.Errorf("token expired")
	}

	return &User{
		ID:       claims.UserID,
		Username: claims.Username,
		Email:    claims.Email,
		Roles:    claims.Roles,
	}, nil
}

// HasAdminRole checks if a user has admin role
func (j *JWTAuthenticator) HasAdminRole(user interface{}) bool {
	u, ok := user.(*User)
	if !ok {
		return false
	}

	for _, role := range u.Roles {
		if role == "admin" || role == "queue-admin" {
			return true
		}
	}

	return false
}

// GenerateToken generates a JWT token for a user
func (j *JWTAuthenticator) GenerateToken(user *User) (string, error) {
	now := time.Now()
	claims := &Claims{
		UserID:   user.ID,
		Username: user.Username,
		Email:    user.Email,
		Roles:    user.Roles,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    j.issuer,
			Audience:  []string{j.audience},
			Subject:   user.ID,
			ExpiresAt: jwt.NewNumericDate(now.Add(24 * time.Hour)),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(j.privateKey)
}

// NoOpAuthenticator is a no-op authenticator for development
type NoOpAuthenticator struct{}

// NewNoOpAuthenticator creates a new no-op authenticator
func NewNoOpAuthenticator() *NoOpAuthenticator {
	return &NoOpAuthenticator{}
}

// ValidateToken always returns a default admin user
func (n *NoOpAuthenticator) ValidateToken(ctx context.Context, token string) (*User, error) {
	return &User{
		ID:       "dev-user",
		Username: "developer",
		Email:    "dev@example.com",
		Roles:    []string{"admin"},
	}, nil
}

// HasAdminRole always returns true
func (n *NoOpAuthenticator) HasAdminRole(user interface{}) bool {
	return true
}

// GenerateToken generates a dummy token
func (n *NoOpAuthenticator) GenerateToken(user *User) (string, error) {
	return "dev-token", nil
}

// APIKeyAuthenticator implements API key-based authentication
type APIKeyAuthenticator struct {
	apiKeys map[string]*User
}

// NewAPIKeyAuthenticator creates a new API key authenticator
func NewAPIKeyAuthenticator(apiKeys map[string]*User) *APIKeyAuthenticator {
	return &APIKeyAuthenticator{
		apiKeys: apiKeys,
	}
}

// ValidateToken validates an API key
func (a *APIKeyAuthenticator) ValidateToken(ctx context.Context, token string) (*User, error) {
	// Remove "Bearer " prefix if present
	token = strings.TrimPrefix(token, "Bearer ")

	user, exists := a.apiKeys[token]
	if !exists {
		return nil, fmt.Errorf("invalid API key")
	}

	return user, nil
}

// HasAdminRole checks if a user has admin role
func (a *APIKeyAuthenticator) HasAdminRole(user interface{}) bool {
	u, ok := user.(*User)
	if !ok {
		return false
	}

	for _, role := range u.Roles {
		if role == "admin" || role == "queue-admin" {
			return true
		}
	}

	return false
}

// GenerateToken is not supported for API key auth
func (a *APIKeyAuthenticator) GenerateToken(user *User) (string, error) {
	return "", fmt.Errorf("token generation not supported for API key auth")
}
