package auth

import (
	"context"
	"fmt"

	"github.com/cucumber/godog"
)

// User store and authentication BDD step implementations

func (ctx *AuthBDDTestContext) iHaveAUserStoreConfigured() error {
	// User store is configured as part of the module
	return nil
}

func (ctx *AuthBDDTestContext) iCreateANewUser() error {
	user := &User{
		ID:    "new-user-123",
		Email: "newuser@example.com",
	}

	err := ctx.service.userStore.CreateUser(context.Background(), user)
	if err != nil {
		ctx.lastError = err
		return nil
	}
	ctx.user = user
	ctx.userID = user.ID
	return nil
}

func (ctx *AuthBDDTestContext) theUserShouldBeStoredSuccessfully() error {
	if ctx.lastError != nil {
		return fmt.Errorf("user creation failed: %v", ctx.lastError)
	}
	return nil
}

func (ctx *AuthBDDTestContext) iShouldBeAbleToRetrieveTheUserByID() error {
	user, err := ctx.service.userStore.GetUser(context.Background(), ctx.userID)
	if err != nil {
		return fmt.Errorf("failed to retrieve user: %v", err)
	}
	if user == nil {
		return fmt.Errorf("user not found")
	}
	return nil
}

func (ctx *AuthBDDTestContext) iHaveAUserWithCredentialsInTheStore() error {
	hashedPassword, err := ctx.service.HashPassword("userpassword123!")
	if err != nil {
		return fmt.Errorf("failed to hash password: %v", err)
	}

	user := &User{
		ID:           "auth-user-123",
		Email:        "authuser@example.com",
		PasswordHash: hashedPassword,
	}

	err = ctx.service.userStore.CreateUser(context.Background(), user)
	if err != nil {
		return fmt.Errorf("failed to create user: %v", err)
	}

	ctx.user = user
	ctx.password = "userpassword123!"
	return nil
}

func (ctx *AuthBDDTestContext) iAuthenticateWithCorrectCredentials() error {
	// Implement authentication using GetUserByEmail and VerifyPassword
	user, err := ctx.service.userStore.GetUserByEmail(context.Background(), ctx.user.Email)
	if err != nil {
		ctx.authError = err
		return nil
	}

	err = ctx.service.VerifyPassword(user.PasswordHash, ctx.password)
	if err != nil {
		ctx.authError = err
		return nil
	}

	ctx.authResult = user
	return nil
}

func (ctx *AuthBDDTestContext) theAuthenticationShouldSucceed() error {
	if ctx.authError != nil {
		return fmt.Errorf("authentication failed: %v", ctx.authError)
	}
	if ctx.authResult == nil {
		return fmt.Errorf("no user returned from authentication")
	}
	return nil
}

func (ctx *AuthBDDTestContext) theUserShouldBeReturned() error {
	if ctx.authResult == nil {
		return fmt.Errorf("no user returned")
	}
	if ctx.authResult.ID != ctx.user.ID {
		return fmt.Errorf("wrong user returned: expected %s, got %s", ctx.user.ID, ctx.authResult.ID)
	}
	return nil
}

func (ctx *AuthBDDTestContext) iAuthenticateWithIncorrectCredentials() error {
	// Implement authentication using GetUserByEmail and VerifyPassword
	user, err := ctx.service.userStore.GetUserByEmail(context.Background(), ctx.user.Email)
	if err != nil {
		ctx.authError = err
		return nil
	}

	err = ctx.service.VerifyPassword(user.PasswordHash, "wrongpassword")
	if err != nil {
		ctx.authError = err
		return nil
	}

	ctx.authResult = user
	return nil
}

func (ctx *AuthBDDTestContext) theAuthenticationShouldFail() error {
	if ctx.authError == nil {
		return fmt.Errorf("authentication should have failed")
	}
	return nil
}

func (ctx *AuthBDDTestContext) anErrorShouldBeReturned() error {
	if ctx.authError == nil {
		return fmt.Errorf("no error returned")
	}
	return nil
}

// User store-specific step registration
func (ctx *AuthBDDTestContext) registerUserStoreSteps(s *godog.ScenarioContext) {
	// User store steps
	s.Step(`^I have a user store configured$`, ctx.iHaveAUserStoreConfigured)
	s.Step(`^I create a new user$`, ctx.iCreateANewUser)
	s.Step(`^the user should be stored successfully$`, ctx.theUserShouldBeStoredSuccessfully)
	s.Step(`^I should be able to retrieve the user by ID$`, ctx.iShouldBeAbleToRetrieveTheUserByID)

	// Authentication steps
	s.Step(`^I have a user with credentials in the store$`, ctx.iHaveAUserWithCredentialsInTheStore)
	s.Step(`^I authenticate with correct credentials$`, ctx.iAuthenticateWithCorrectCredentials)
	s.Step(`^the authentication should succeed$`, ctx.theAuthenticationShouldSucceed)
	s.Step(`^the user should be returned$`, ctx.theUserShouldBeReturned)
	s.Step(`^I authenticate with incorrect credentials$`, ctx.iAuthenticateWithIncorrectCredentials)
	s.Step(`^the authentication should fail$`, ctx.theAuthenticationShouldFail)
	s.Step(`^an error should be returned$`, ctx.anErrorShouldBeReturned)
}
