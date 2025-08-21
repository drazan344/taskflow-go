package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/drazan344/taskflow-go/internal/config"
	"github.com/drazan344/taskflow-go/internal/models"
)

// TokenType represents the type of JWT token
type TokenType string

const (
	AccessToken  TokenType = "access"
	RefreshToken TokenType = "refresh"
)

// Claims represents the JWT claims
type Claims struct {
	UserID     uuid.UUID        `json:"user_id"`
	TenantID   uuid.UUID        `json:"tenant_id"`
	Email      string           `json:"email"`
	Role       models.UserRole  `json:"role"`
	TokenType  TokenType        `json:"token_type"`
	SessionID  uuid.UUID        `json:"session_id"`
	jwt.RegisteredClaims
}

// JWTService handles JWT token operations
type JWTService struct {
	config *config.Config
}

// NewJWTService creates a new JWT service
func NewJWTService(cfg *config.Config) *JWTService {
	return &JWTService{config: cfg}
}

// GenerateTokenPair generates both access and refresh tokens
func (s *JWTService) GenerateTokenPair(user *models.User, sessionID uuid.UUID) (string, string, error) {
	// Generate access token
	accessToken, err := s.generateToken(user, sessionID, AccessToken, s.config.JWT.Expiry)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate access token: %w", err)
	}

	// Generate refresh token
	refreshToken, err := s.generateToken(user, sessionID, RefreshToken, s.config.JWT.RefreshExpiry)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return accessToken, refreshToken, nil
}

// generateToken generates a JWT token
func (s *JWTService) generateToken(user *models.User, sessionID uuid.UUID, tokenType TokenType, expiry time.Duration) (string, error) {
	now := time.Now()
	expirationTime := now.Add(expiry)

	claims := &Claims{
		UserID:    user.ID,
		TenantID:  user.TenantID,
		Email:     user.Email,
		Role:      user.Role,
		TokenType: tokenType,
		SessionID: sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "taskflow-go",
			Subject:   user.ID.String(),
			ID:        uuid.New().String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	var secret string
	if tokenType == AccessToken {
		secret = s.config.JWT.Secret
	} else {
		secret = s.config.JWT.RefreshSecret
	}

	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// ValidateToken validates a JWT token and returns the claims
func (s *JWTService) ValidateToken(tokenString string, tokenType TokenType) (*Claims, error) {
	var secret string
	if tokenType == AccessToken {
		secret = s.config.JWT.Secret
	} else {
		secret = s.config.JWT.RefreshSecret
	}

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	// Verify token type matches expected
	if claims.TokenType != tokenType {
		return nil, fmt.Errorf("invalid token type: expected %s, got %s", tokenType, claims.TokenType)
	}

	// Check if token is expired
	if claims.ExpiresAt != nil && time.Now().After(claims.ExpiresAt.Time) {
		return nil, fmt.Errorf("token has expired")
	}

	// Check if token is not yet valid
	if claims.NotBefore != nil && time.Now().Before(claims.NotBefore.Time) {
		return nil, fmt.Errorf("token not yet valid")
	}

	return claims, nil
}

// RefreshTokens generates new token pair using refresh token
func (s *JWTService) RefreshTokens(refreshTokenString string, user *models.User, sessionID uuid.UUID) (string, string, error) {
	// Validate the refresh token
	claims, err := s.ValidateToken(refreshTokenString, RefreshToken)
	if err != nil {
		return "", "", fmt.Errorf("invalid refresh token: %w", err)
	}

	// Verify the refresh token belongs to the user and session
	if claims.UserID != user.ID || claims.SessionID != sessionID {
		return "", "", fmt.Errorf("refresh token does not match user or session")
	}

	// Generate new token pair
	return s.GenerateTokenPair(user, sessionID)
}

// ExtractTokenFromHeader extracts JWT token from Authorization header
func ExtractTokenFromHeader(authHeader string) (string, error) {
	if authHeader == "" {
		return "", fmt.Errorf("authorization header is empty")
	}

	const bearerPrefix = "Bearer "
	if len(authHeader) < len(bearerPrefix) || authHeader[:len(bearerPrefix)] != bearerPrefix {
		return "", fmt.Errorf("invalid authorization header format")
	}

	token := authHeader[len(bearerPrefix):]
	if token == "" {
		return "", fmt.Errorf("token is empty")
	}

	return token, nil
}

// GetUserIDFromToken extracts user ID from token claims
func GetUserIDFromToken(claims *Claims) uuid.UUID {
	return claims.UserID
}

// GetTenantIDFromToken extracts tenant ID from token claims
func GetTenantIDFromToken(claims *Claims) uuid.UUID {
	return claims.TenantID
}

// GetUserRoleFromToken extracts user role from token claims
func GetUserRoleFromToken(claims *Claims) models.UserRole {
	return claims.Role
}

// GetSessionIDFromToken extracts session ID from token claims
func GetSessionIDFromToken(claims *Claims) uuid.UUID {
	return claims.SessionID
}

// IsTokenExpired checks if a token is expired
func IsTokenExpired(claims *Claims) bool {
	if claims.ExpiresAt == nil {
		return false
	}
	return time.Now().After(claims.ExpiresAt.Time)
}

// GetTokenRemainingTime returns the remaining time until token expires
func GetTokenRemainingTime(claims *Claims) time.Duration {
	if claims.ExpiresAt == nil {
		return 0
	}
	remaining := time.Until(claims.ExpiresAt.Time)
	if remaining < 0 {
		return 0
	}
	return remaining
}