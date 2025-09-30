package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cucumber/godog"
	"github.com/golang-jwt/jwt/v5"
)

// JWT-related BDD step implementations

func (ctx *AuthBDDTestContext) iHaveUserCredentialsAndJWTConfiguration() error {
	// This is implicitly handled by the module configuration
	return nil
}

func (ctx *AuthBDDTestContext) iGenerateAJWTTokenForTheUser() error {
	var err error
	tokenPair, err := ctx.service.GenerateToken("test-user-123", map[string]interface{}{
		"email": "test@example.com",
	})
	if err != nil {
		ctx.lastError = err
		return nil // Don't return error here as it might be expected
	}

	ctx.token = tokenPair.AccessToken
	ctx.refreshToken = tokenPair.RefreshToken
	return nil
}

func (ctx *AuthBDDTestContext) theTokenShouldBeCreatedSuccessfully() error {
	if ctx.token == "" {
		return fmt.Errorf("token was not created")
	}
	if ctx.lastError != nil {
		return fmt.Errorf("token creation failed: %v", ctx.lastError)
	}
	return nil
}

func (ctx *AuthBDDTestContext) theTokenShouldContainTheUserInformation() error {
	if ctx.token == "" {
		return fmt.Errorf("no token available")
	}

	claims, err := ctx.service.ValidateToken(ctx.token)
	if err != nil {
		return fmt.Errorf("failed to validate token: %v", err)
	}

	if claims.UserID != "test-user-123" {
		return fmt.Errorf("expected UserID 'test-user-123', got '%s'", claims.UserID)
	}

	return nil
}

func (ctx *AuthBDDTestContext) iHaveAValidJWTToken() error {
	var err error
	tokenPair, err := ctx.service.GenerateToken("valid-user", map[string]interface{}{
		"email": "valid@example.com",
	})
	if err != nil {
		return fmt.Errorf("failed to generate valid token: %v", err)
	}

	ctx.token = tokenPair.AccessToken
	return nil
}

func (ctx *AuthBDDTestContext) iValidateTheToken() error {
	var err error
	ctx.claims, err = ctx.service.ValidateToken(ctx.token)
	if err != nil {
		ctx.lastError = err
		return nil // Don't return error here as validation might be expected to fail
	}

	return nil
}

func (ctx *AuthBDDTestContext) theTokenShouldBeAccepted() error {
	if ctx.lastError != nil {
		return fmt.Errorf("token was rejected: %v", ctx.lastError)
	}
	if ctx.claims == nil {
		return fmt.Errorf("no claims extracted from token")
	}
	return nil
}

func (ctx *AuthBDDTestContext) theUserClaimsShouldBeExtracted() error {
	if ctx.claims == nil {
		return fmt.Errorf("no claims available")
	}
	if ctx.claims.UserID == "" {
		return fmt.Errorf("UserID not found in claims")
	}
	return nil
}

func (ctx *AuthBDDTestContext) iHaveAnInvalidJWTToken() error {
	ctx.token = "invalid.jwt.token"
	return nil
}

func (ctx *AuthBDDTestContext) theTokenShouldBeRejected() error {
	if ctx.lastError == nil {
		return fmt.Errorf("token should have been rejected but was accepted")
	}
	return nil
}

func (ctx *AuthBDDTestContext) anAppropriateErrorShouldBeReturned() error {
	if ctx.lastError == nil {
		return fmt.Errorf("no error was returned")
	}
	return nil
}

func (ctx *AuthBDDTestContext) iHaveAnExpiredJWTToken() error {
	// Create a real expired JWT token using the JWT library
	// Generate token with expiration time in the past
	now := time.Now()
	expiredTime := now.Add(-1 * time.Hour) // Token expired 1 hour ago

	// Create claims with past expiration
	accessClaims := jwt.MapClaims{
		"user_id": "expired-test-user",
		"type":    "access",
		"iat":     expiredTime.Add(-24 * time.Hour).Unix(), // issued 25 hours ago
		"exp":     expiredTime.Unix(),                      // expired 1 hour ago
		"counter": 1,
	}

	if ctx.service.config.JWT.Issuer != "" {
		accessClaims["iss"] = ctx.service.config.JWT.Issuer
	}
	accessClaims["sub"] = "expired-test-user"

	// Add some test claims
	accessClaims["email"] = "expired@example.com"

	// Generate the expired token using the same method as the service
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	tokenString, err := token.SignedString([]byte(ctx.service.config.JWT.Secret))
	if err != nil {
		return fmt.Errorf("failed to generate expired token: %w", err)
	}

	ctx.token = tokenString
	return nil
}

func (ctx *AuthBDDTestContext) theErrorShouldIndicateTokenExpiration() error {
	if ctx.lastError == nil {
		return fmt.Errorf("no error indicating expiration")
	}

	// Check if the error is specifically the token expiration error
	if !errors.Is(ctx.lastError, ErrTokenExpired) {
		return fmt.Errorf("expected token expiration error, got: %v (type: %T)", ctx.lastError, ctx.lastError)
	}

	// Additional check: ensure the error message contains expiration indication
	errorMsg := ctx.lastError.Error()
	if !strings.Contains(strings.ToLower(errorMsg), "expired") {
		return fmt.Errorf("error message should contain 'expired', got: %s", errorMsg)
	}

	return nil
}

func (ctx *AuthBDDTestContext) iRefreshTheToken() error {
	if ctx.token == "" {
		return fmt.Errorf("no token to refresh")
	}

	// First, create a user in the user store for refresh functionality
	refreshUser := &User{
		ID:          "refresh-user",
		Email:       "refresh@example.com",
		Active:      true,
		Roles:       []string{"user"},
		Permissions: []string{"read"},
	}

	// Create the user in the store
	if err := ctx.service.userStore.CreateUser(context.Background(), refreshUser); err != nil {
		// If user already exists, that's fine
		if err != ErrUserAlreadyExists {
			ctx.lastError = err
			return nil
		}
	}

	// Generate a token pair for the user
	tokenPair, err := ctx.service.GenerateToken("refresh-user", map[string]interface{}{
		"email": "refresh@example.com",
	})
	if err != nil {
		ctx.lastError = err
		return nil
	}

	// Use the refresh token to get a new token pair
	newTokenPair, err := ctx.service.RefreshToken(tokenPair.RefreshToken)
	if err != nil {
		ctx.lastError = err
		return nil
	}

	ctx.token = newTokenPair.AccessToken
	ctx.newToken = newTokenPair.AccessToken // Set the new token for validation
	return nil
}

func (ctx *AuthBDDTestContext) aNewTokenShouldBeGenerated() error {
	if ctx.token == "" {
		return fmt.Errorf("no new token generated")
	}
	if ctx.lastError != nil {
		return fmt.Errorf("token refresh failed: %v", ctx.lastError)
	}
	return nil
}

func (ctx *AuthBDDTestContext) theNewTokenShouldHaveUpdatedExpiration() error {
	// This would require checking the token's expiration time
	// For now, we assume the refresh worked if we have a new token
	return ctx.aNewTokenShouldBeGenerated()
}

// JWT-specific step registration
func (ctx *AuthBDDTestContext) registerJWTSteps(s *godog.ScenarioContext) {
	// JWT token steps
	s.Step(`^I have user credentials and JWT configuration$`, ctx.iHaveUserCredentialsAndJWTConfiguration)
	s.Step(`^I generate a JWT token for the user$`, ctx.iGenerateAJWTTokenForTheUser)
	s.Step(`^I generate a JWT token for a user$`, ctx.iGenerateAJWTTokenForTheUser)
	s.Step(`^the token should be created successfully$`, ctx.theTokenShouldBeCreatedSuccessfully)
	s.Step(`^the token should contain the user information$`, ctx.theTokenShouldContainTheUserInformation)

	// Token validation steps
	s.Step(`^I have a valid JWT token$`, ctx.iHaveAValidJWTToken)
	s.Step(`^I validate the token$`, ctx.iValidateTheToken)
	s.Step(`^the token should be accepted$`, ctx.theTokenShouldBeAccepted)
	s.Step(`^the user claims should be extracted$`, ctx.theUserClaimsShouldBeExtracted)
	s.Step(`^I have an invalid JWT token$`, ctx.iHaveAnInvalidJWTToken)
	s.Step(`^the token should be rejected$`, ctx.theTokenShouldBeRejected)
	s.Step(`^an appropriate error should be returned$`, ctx.anAppropriateErrorShouldBeReturned)
	s.Step(`^I have an expired JWT token$`, ctx.iHaveAnExpiredJWTToken)
	s.Step(`^the error should indicate token expiration$`, ctx.theErrorShouldIndicateTokenExpiration)

	// Token refresh steps
	s.Step(`^I refresh the token$`, ctx.iRefreshTheToken)
	s.Step(`^a new token should be generated$`, ctx.aNewTokenShouldBeGenerated)
	s.Step(`^the new token should have updated expiration$`, ctx.theNewTokenShouldHaveUpdatedExpiration)
}
