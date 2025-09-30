package auth

import (
	"fmt"

	"github.com/cucumber/godog"
)

// Session management BDD step implementations

func (ctx *AuthBDDTestContext) iHaveAUserIdentifier() error {
	ctx.userID = "session-user-123"
	return nil
}

func (ctx *AuthBDDTestContext) iCreateANewSessionForTheUser() error {
	var err error
	ctx.session, err = ctx.service.CreateSession(ctx.userID, map[string]interface{}{
		"created_by": "bdd_test",
	})
	if err != nil {
		ctx.lastError = err
		return nil
	}
	if ctx.session != nil {
		ctx.sessionID = ctx.session.ID
	}
	return nil
}

func (ctx *AuthBDDTestContext) theSessionShouldBeCreatedSuccessfully() error {
	if ctx.session == nil {
		return fmt.Errorf("session was not created")
	}
	if ctx.lastError != nil {
		return fmt.Errorf("session creation failed: %v", ctx.lastError)
	}
	return nil
}

func (ctx *AuthBDDTestContext) theSessionShouldHaveAUniqueID() error {
	if ctx.session == nil {
		return fmt.Errorf("no session available")
	}
	if ctx.session.ID == "" {
		return fmt.Errorf("session ID is empty")
	}
	return nil
}

func (ctx *AuthBDDTestContext) iHaveAnExistingUserSession() error {
	ctx.userID = "existing-user-123"
	var err error
	ctx.session, err = ctx.service.CreateSession(ctx.userID, map[string]interface{}{
		"test": "existing_session",
	})
	if err != nil {
		return fmt.Errorf("failed to create existing session: %v", err)
	}
	ctx.sessionID = ctx.session.ID
	return nil
}

func (ctx *AuthBDDTestContext) iRetrieveTheSessionByID() error {
	var err error
	ctx.session, err = ctx.service.GetSession(ctx.sessionID)
	if err != nil {
		ctx.lastError = err
		return nil
	}
	return nil
}

func (ctx *AuthBDDTestContext) theSessionShouldBeFound() error {
	if ctx.session == nil {
		return fmt.Errorf("session was not found")
	}
	if ctx.lastError != nil {
		return fmt.Errorf("session retrieval failed: %v", ctx.lastError)
	}
	return nil
}

func (ctx *AuthBDDTestContext) theSessionDataShouldMatch() error {
	if ctx.session == nil {
		return fmt.Errorf("no session to check")
	}
	if ctx.session.ID != ctx.sessionID {
		return fmt.Errorf("session ID mismatch: expected %s, got %s", ctx.sessionID, ctx.session.ID)
	}
	return nil
}

func (ctx *AuthBDDTestContext) iDeleteTheSession() error {
	err := ctx.service.DeleteSession(ctx.sessionID)
	if err != nil {
		ctx.lastError = err
		return nil
	}
	return nil
}

func (ctx *AuthBDDTestContext) theSessionShouldBeRemoved() error {
	if ctx.lastError != nil {
		return fmt.Errorf("session deletion failed: %v", ctx.lastError)
	}
	return nil
}

func (ctx *AuthBDDTestContext) subsequentRetrievalShouldFail() error {
	session, err := ctx.service.GetSession(ctx.sessionID)
	if err == nil && session != nil {
		return fmt.Errorf("session should have been deleted but was found")
	}
	return nil
}

// Session-specific step registration
func (ctx *AuthBDDTestContext) registerSessionSteps(s *godog.ScenarioContext) {
	// Session management steps
	s.Step(`^I have a user identifier$`, ctx.iHaveAUserIdentifier)
	s.Step(`^I create a new session for the user$`, ctx.iCreateANewSessionForTheUser)
	s.Step(`^the session should be created successfully$`, ctx.theSessionShouldBeCreatedSuccessfully)
	s.Step(`^the session should have a unique ID$`, ctx.theSessionShouldHaveAUniqueID)
	s.Step(`^I have an existing user session$`, ctx.iHaveAnExistingUserSession)
	s.Step(`^I retrieve the session by ID$`, ctx.iRetrieveTheSessionByID)
	s.Step(`^the session should be found$`, ctx.theSessionShouldBeFound)
	s.Step(`^the session data should match$`, ctx.theSessionDataShouldMatch)
	s.Step(`^I delete the session$`, ctx.iDeleteTheSession)
	s.Step(`^the session should be removed$`, ctx.theSessionShouldBeRemoved)
	s.Step(`^subsequent retrieval should fail$`, ctx.subsequentRetrievalShouldFail)
}
