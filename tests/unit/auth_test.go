package unit

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/drazan344/taskflow-go/internal/auth"
	"github.com/drazan344/taskflow-go/internal/config"
	"github.com/drazan344/taskflow-go/internal/models"
	"github.com/drazan344/taskflow-go/tests/fixtures"
)

func TestJWTService(t *testing.T) {
	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret:        "test-secret-key-for-testing-only",
			RefreshSecret: "test-refresh-secret-key-for-testing",
			Expiry:        15 * time.Minute,
			RefreshExpiry: 7 * 24 * time.Hour,
		},
	}

	jwtService := auth.NewJWTService(cfg)

	t.Run("Generate token pair", func(t *testing.T) {
		user := &models.User{
			TenantModel: models.TenantModel{TenantID: uuid.New()},
			Email:       "test@example.com",
			Role:        models.UserRoleAdmin,
		}
		user.ID = uuid.New()
		sessionID := uuid.New()

		accessToken, refreshToken, err := jwtService.GenerateTokenPair(user, sessionID)
		assert.NoError(t, err)
		assert.NotEmpty(t, accessToken)
		assert.NotEmpty(t, refreshToken)

		// Verify access token
		accessClaims, err := jwtService.ValidateToken(accessToken, auth.AccessToken)
		assert.NoError(t, err)
		assert.Equal(t, user.ID, accessClaims.UserID)
		assert.Equal(t, user.TenantID, accessClaims.TenantID)
		assert.Equal(t, user.Role, accessClaims.Role)
		assert.Equal(t, auth.AccessToken, accessClaims.TokenType)

		// Verify refresh token
		refreshClaims, err := jwtService.ValidateToken(refreshToken, auth.RefreshToken)
		assert.NoError(t, err)
		assert.Equal(t, user.ID, refreshClaims.UserID)
		assert.Equal(t, user.TenantID, refreshClaims.TenantID)
		assert.Equal(t, sessionID, refreshClaims.SessionID)
		assert.Equal(t, auth.RefreshToken, refreshClaims.TokenType)
	})

	t.Run("Invalid token", func(t *testing.T) {
		_, err := jwtService.ValidateToken("invalid-token", auth.AccessToken)
		assert.Error(t, err)

		_, err = jwtService.ValidateToken("invalid-token", auth.RefreshToken)
		assert.Error(t, err)
	})

	t.Run("Expired token", func(t *testing.T) {
		// Create token with short expiration
		shortCfg := &config.Config{
			JWT: config.JWTConfig{
				Secret:        "test-secret-key-for-testing-only",
				RefreshSecret: "test-refresh-secret-key-for-testing",
				Expiry:        1 * time.Nanosecond,
				RefreshExpiry: 7 * 24 * time.Hour,
			},
		}
		shortJWTService := auth.NewJWTService(shortCfg)

		user := &models.User{
			TenantModel: models.TenantModel{TenantID: uuid.New()},
			Email:       "test@example.com",
			Role:        models.UserRoleAdmin,
		}
		user.ID = uuid.New()
		sessionID := uuid.New()

		accessToken, _, err := shortJWTService.GenerateTokenPair(user, sessionID)
		assert.NoError(t, err)

		// Wait a bit to ensure expiration
		time.Sleep(time.Millisecond)

		_, err = shortJWTService.ValidateToken(accessToken, auth.AccessToken)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "expired")
	})

	t.Run("Wrong token type", func(t *testing.T) {
		user := &models.User{
			TenantModel: models.TenantModel{TenantID: uuid.New()},
			Email:       "test@example.com",
			Role:        models.UserRoleAdmin,
		}
		user.ID = uuid.New()
		sessionID := uuid.New()

		accessToken, _, err := jwtService.GenerateTokenPair(user, sessionID)
		assert.NoError(t, err)

		// Try to validate access token as refresh token (this should fail because of different secret)
		_, err = jwtService.ValidateToken(accessToken, auth.RefreshToken)
		assert.Error(t, err)
		// The error could be either signature invalid or token type mismatch depending on validation order
	})
}

func TestAuthService(t *testing.T) {
	db, err := fixtures.NewTestDB()
	require.NoError(t, err)
	defer db.Close()

	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret:        "test-secret-key-for-testing-only",
			RefreshSecret: "test-refresh-secret-key-for-testing",
			Expiry:        15 * time.Minute,
			RefreshExpiry: 7 * 24 * time.Hour,
		},
	}

	authService := auth.NewService(db.DB, cfg)

	t.Run("Register user", func(t *testing.T) {
		req := &auth.RegisterRequest{
			Email:      "register@test.com",
			FirstName:  "Test",
			LastName:   "Register",
			Password:   "password123",
			TenantName: "Test Tenant",
			TenantSlug: "test-tenant",
		}

		resp, err := authService.Register(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, req.Email, resp.User.Email)
		assert.Equal(t, req.FirstName, resp.User.FirstName)
		assert.NotEmpty(t, resp.AccessToken)
		assert.NotEmpty(t, resp.RefreshToken)
	})

	t.Run("Login user", func(t *testing.T) {
		_, err := db.SeedTestData()
		require.NoError(t, err)

		// First register a user
		registerReq := &auth.RegisterRequest{
			Email:      "login@test.com",
			FirstName:  "Test",
			LastName:   "Login",
			Password:   "password123",
			TenantName: "Login Test Tenant",
			TenantSlug: "login-test-tenant",
		}

		_, err = authService.Register(registerReq)
		require.NoError(t, err)

		// Now login
		loginReq := &auth.LoginRequest{
			Email:    "login@test.com",
			Password: "password123",
		}

		resp, err := authService.Login(loginReq, "127.0.0.1", "test-agent")
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, loginReq.Email, resp.User.Email)
		assert.NotEmpty(t, resp.AccessToken)
		assert.NotEmpty(t, resp.RefreshToken)
	})

	t.Run("Login with wrong password", func(t *testing.T) {
		_, err := db.SeedTestData()
		require.NoError(t, err)

		// First register a user
		registerReq := &auth.RegisterRequest{
			Email:      "wrong@test.com",
			FirstName:  "Test",
			LastName:   "Wrong",
			Password:   "password123",
			TenantName: "Wrong Test Tenant",
			TenantSlug: "wrong-test-tenant",
		}

		_, err = authService.Register(registerReq)
		require.NoError(t, err)

		// Try login with wrong password
		loginReq := &auth.LoginRequest{
			Email:    "wrong@test.com",
			Password: "wrongpassword",
		}

		_, err = authService.Login(loginReq, "127.0.0.1", "test-agent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid credentials")
	})

	t.Run("Login non-existent user", func(t *testing.T) {
		_, err := db.SeedTestData()
		require.NoError(t, err)

		loginReq := &auth.LoginRequest{
			Email:    "nonexistent@test.com",
			Password: "password123",
		}

		_, err = authService.Login(loginReq, "127.0.0.1", "test-agent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid credentials")
	})

	t.Run("RefreshTokens", func(t *testing.T) {
		_, err := db.SeedTestData()
		require.NoError(t, err)

		// First register a user
		registerReq := &auth.RegisterRequest{
			Email:      "refresh@test.com",
			FirstName:  "Test",
			LastName:   "Refresh",
			Password:   "password123",
			TenantName: "Refresh Test Tenant",
			TenantSlug: "refresh-test-tenant",
		}

		registerResp, err := authService.Register(registerReq)
		require.NoError(t, err)

		// Use refresh token to get new tokens
		refreshReq := &auth.RefreshTokenRequest{
			RefreshToken: registerResp.RefreshToken,
		}

		refreshResp, err := authService.RefreshTokens(refreshReq)
		assert.NoError(t, err)
		assert.NotNil(t, refreshResp)
		assert.NotEmpty(t, refreshResp.AccessToken)
		assert.NotEmpty(t, refreshResp.RefreshToken)
		assert.NotEqual(t, registerResp.AccessToken, refreshResp.AccessToken)
	})

	t.Run("Refresh with invalid token", func(t *testing.T) {
		refreshReq := &auth.RefreshTokenRequest{
			RefreshToken: "invalid-refresh-token",
		}

		_, err := authService.RefreshTokens(refreshReq)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid refresh token")
	})
}

func TestPasswordHashing(t *testing.T) {
	password := "testpassword123"

	t.Run("Hash password", func(t *testing.T) {
		user := &models.User{}
		err := user.SetPassword(password)
		assert.NoError(t, err)
		assert.NotEmpty(t, user.Password)
		assert.NotEqual(t, password, user.Password)
	})

	t.Run("Check password", func(t *testing.T) {
		user := &models.User{}
		err := user.SetPassword(password)
		require.NoError(t, err)

		// Correct password
		assert.True(t, user.CheckPassword(password))

		// Wrong password
		assert.False(t, user.CheckPassword("wrongpassword"))
	})
}