package auth

import (
	"fmt"

	"github.com/cucumber/godog"
)

// Password-related BDD step implementations

func (ctx *AuthBDDTestContext) iHaveAPlainTextPassword() error {
	ctx.password = "MySecurePassword123!"
	return nil
}

func (ctx *AuthBDDTestContext) iHashThePasswordUsingBcrypt() error {
	var err error
	ctx.hashedPassword, err = ctx.service.HashPassword(ctx.password)
	if err != nil {
		ctx.lastError = err
		return nil
	}
	return nil
}

func (ctx *AuthBDDTestContext) thePasswordShouldBeHashedSuccessfully() error {
	if ctx.hashedPassword == "" {
		return fmt.Errorf("password was not hashed")
	}
	if ctx.lastError != nil {
		return fmt.Errorf("password hashing failed: %v", ctx.lastError)
	}
	return nil
}

func (ctx *AuthBDDTestContext) theHashShouldBeDifferentFromTheOriginalPassword() error {
	if ctx.hashedPassword == ctx.password {
		return fmt.Errorf("hash is the same as original password")
	}
	return nil
}

func (ctx *AuthBDDTestContext) iHaveAPasswordAndItsHash() error {
	ctx.password = "TestPassword123!"
	var err error
	ctx.hashedPassword, err = ctx.service.HashPassword(ctx.password)
	if err != nil {
		return fmt.Errorf("failed to hash password: %v", err)
	}
	return nil
}

func (ctx *AuthBDDTestContext) iVerifyThePasswordAgainstTheHash() error {
	err := ctx.service.VerifyPassword(ctx.hashedPassword, ctx.password)
	ctx.verifyResult = (err == nil)
	return nil
}

func (ctx *AuthBDDTestContext) theVerificationShouldSucceed() error {
	if !ctx.verifyResult {
		return fmt.Errorf("password verification failed")
	}
	return nil
}

func (ctx *AuthBDDTestContext) iHaveAPasswordAndADifferentHash() error {
	ctx.password = "CorrectPassword123!"
	wrongPassword := "WrongPassword123!"
	var err error
	ctx.hashedPassword, err = ctx.service.HashPassword(wrongPassword)
	if err != nil {
		return fmt.Errorf("failed to hash wrong password: %v", err)
	}
	return nil
}

func (ctx *AuthBDDTestContext) theVerificationShouldFail() error {
	if ctx.verifyResult {
		return fmt.Errorf("password verification should have failed")
	}
	return nil
}

func (ctx *AuthBDDTestContext) iHaveAStrongPassword() error {
	ctx.password = "StrongPassword123!@#"
	return nil
}

func (ctx *AuthBDDTestContext) iValidateThePasswordStrength() error {
	ctx.strengthError = ctx.service.ValidatePasswordStrength(ctx.password)
	return nil
}

func (ctx *AuthBDDTestContext) thePasswordShouldBeAccepted() error {
	if ctx.strengthError != nil {
		return fmt.Errorf("strong password was rejected: %v", ctx.strengthError)
	}
	return nil
}

func (ctx *AuthBDDTestContext) noStrengthErrorsShouldBeReported() error {
	if ctx.strengthError != nil {
		return fmt.Errorf("unexpected strength error: %v", ctx.strengthError)
	}
	return nil
}

func (ctx *AuthBDDTestContext) iHaveAWeakPassword() error {
	ctx.password = "weak" // Too short, no uppercase, no numbers, no special chars
	return nil
}

func (ctx *AuthBDDTestContext) thePasswordShouldBeRejected() error {
	if ctx.strengthError == nil {
		return fmt.Errorf("weak password should have been rejected")
	}
	return nil
}

func (ctx *AuthBDDTestContext) appropriateStrengthErrorsShouldBeReported() error {
	if ctx.strengthError == nil {
		return fmt.Errorf("no strength errors reported")
	}
	return nil
}

// Password-specific step registration
func (ctx *AuthBDDTestContext) registerPasswordSteps(s *godog.ScenarioContext) {
	// Password hashing steps
	s.Step(`^I have a plain text password$`, ctx.iHaveAPlainTextPassword)
	s.Step(`^I hash the password using bcrypt$`, ctx.iHashThePasswordUsingBcrypt)
	s.Step(`^the password should be hashed successfully$`, ctx.thePasswordShouldBeHashedSuccessfully)
	s.Step(`^the hash should be different from the original password$`, ctx.theHashShouldBeDifferentFromTheOriginalPassword)

	// Password verification steps
	s.Step(`^I have a password and its hash$`, ctx.iHaveAPasswordAndItsHash)
	s.Step(`^I verify the password against the hash$`, ctx.iVerifyThePasswordAgainstTheHash)
	s.Step(`^the verification should succeed$`, ctx.theVerificationShouldSucceed)
	s.Step(`^I have a password and a different hash$`, ctx.iHaveAPasswordAndADifferentHash)
	s.Step(`^the verification should fail$`, ctx.theVerificationShouldFail)

	// Password strength steps
	s.Step(`^I have a strong password$`, ctx.iHaveAStrongPassword)
	s.Step(`^I validate the password strength$`, ctx.iValidateThePasswordStrength)
	s.Step(`^the password should be accepted$`, ctx.thePasswordShouldBeAccepted)
	s.Step(`^no strength errors should be reported$`, ctx.noStrengthErrorsShouldBeReported)
	s.Step(`^I have a weak password$`, ctx.iHaveAWeakPassword)
	s.Step(`^the password should be rejected$`, ctx.thePasswordShouldBeRejected)
	s.Step(`^appropriate strength errors should be reported$`, ctx.appropriateStrengthErrorsShouldBeReported)
}
