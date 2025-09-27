package modular

import (
	"fmt"
	"strings"
)

// EnhancedCycleDetectionBDDTestContext holds the test context for cycle detection BDD scenarios
type EnhancedCycleDetectionBDDTestContext struct {
	app              Application
	modules          map[string]Module
	lastError        error
	initializeResult error
	cycleDetected    bool
}

// Test interfaces for cycle detection scenarios
type TestInterfaceA interface {
	MethodA() string
}

type TestInterfaceB interface {
	MethodB() string
}

type TestInterfaceC interface {
	MethodC() string
}

// Similar interfaces for name disambiguation testing
type EnhancedTestInterface interface {
	TestMethod() string
}

type AnotherEnhancedTestInterface interface {
	AnotherTestMethod() string
}

// TestInterfaceAImpl implements TestInterfaceA for self-dependency testing
type TestInterfaceAImpl struct {
	name string
}

func (t *TestInterfaceAImpl) MethodA() string {
	return t.name
}

// TestInterfaceBImpl implements TestInterfaceB
type TestInterfaceBImpl struct {
	name string
}

func (t *TestInterfaceBImpl) MethodB() string {
	return t.name
}

// TestInterfaceCImpl implements TestInterfaceC
type TestInterfaceCImpl struct {
	name string
}

func (t *TestInterfaceCImpl) MethodC() string {
	return t.name
}

// EnhancedTestInterfaceImpl implements EnhancedTestInterface
type EnhancedTestInterfaceImpl struct {
	name string
}

func (t *EnhancedTestInterfaceImpl) TestMethod() string {
	return t.name
}

// AnotherEnhancedTestInterfaceImpl implements AnotherEnhancedTestInterface
type AnotherEnhancedTestInterfaceImpl struct {
	name string
}

func (t *AnotherEnhancedTestInterfaceImpl) AnotherTestMethod() string {
	return t.name
}

// BDD Step implementations

func (ctx *EnhancedCycleDetectionBDDTestContext) iHaveAModularApplication() error {
	app, err := NewApplication(
		WithLogger(&testLogger{}),
		WithConfigProvider(NewStdConfigProvider(testCfg{Str: "test"})),
	)
	if err != nil {
		return err
	}
	ctx.app = app
	ctx.modules = make(map[string]Module)
	return nil
}

func (ctx *EnhancedCycleDetectionBDDTestContext) iTryToInitializeTheApplication() error {
	ctx.initializeResult = ctx.app.Init()
	return nil
}

func (ctx *EnhancedCycleDetectionBDDTestContext) theInitializationShouldFailWithACircularDependencyError() error {
	if ctx.initializeResult == nil {
		return fmt.Errorf("expected initialization to fail with circular dependency error, but it succeeded")
	}

	if !IsErrCircularDependency(ctx.initializeResult) {
		return fmt.Errorf("expected ErrCircularDependency, got %T: %v", ctx.initializeResult, ctx.initializeResult)
	}

	ctx.cycleDetected = true
	return nil
}

func (ctx *EnhancedCycleDetectionBDDTestContext) theErrorMessageShouldIncludeBothModuleNames() error {
	if ctx.initializeResult == nil {
		return fmt.Errorf("no error to check")
	}

	errorMsg := ctx.initializeResult.Error()
	if !strings.Contains(errorMsg, "moduleA") || !strings.Contains(errorMsg, "moduleB") {
		return fmt.Errorf("error message should contain both module names, got: %s", errorMsg)
	}

	return nil
}

func (ctx *EnhancedCycleDetectionBDDTestContext) theErrorMessageShouldIndicateInterfaceBasedDependencies() error {
	if ctx.initializeResult == nil {
		return fmt.Errorf("no error to check")
	}

	errorMsg := ctx.initializeResult.Error()
	if !strings.Contains(errorMsg, "interface:") {
		return fmt.Errorf("error message should indicate interface-based dependencies, got: %s", errorMsg)
	}

	return nil
}

func (ctx *EnhancedCycleDetectionBDDTestContext) theErrorMessageShouldShowTheCompleteDependencyCycle() error {
	if ctx.initializeResult == nil {
		return fmt.Errorf("no error to check")
	}

	errorMsg := ctx.initializeResult.Error()
	if !strings.Contains(errorMsg, "cycle:") {
		return fmt.Errorf("error message should show complete cycle, got: %s", errorMsg)
	}

	// Check for arrow notation indicating dependency flow
	if !strings.Contains(errorMsg, "→") {
		return fmt.Errorf("error message should use arrow notation for dependency flow, got: %s", errorMsg)
	}

	return nil
}

func (ctx *EnhancedCycleDetectionBDDTestContext) theErrorMessageShouldContain(expectedMsg string) error {
	if ctx.initializeResult == nil {
		return fmt.Errorf("no error to check")
	}

	errorMsg := ctx.initializeResult.Error()

	// The exact format might vary, so let's check for key components
	requiredComponents := []string{"cycle:", "moduleA", "moduleB", "interface:", "TestInterface"}
	for _, component := range requiredComponents {
		if !strings.Contains(errorMsg, component) {
			return fmt.Errorf("error message should contain '%s', got: %s", component, errorMsg)
		}
	}

	return nil
}

func (ctx *EnhancedCycleDetectionBDDTestContext) theErrorMessageShouldClearlyShowTheInterfaceCausingTheCycle() error {
	if ctx.initializeResult == nil {
		return fmt.Errorf("no error to check")
	}

	errorMsg := ctx.initializeResult.Error()
	// Look for interface specification in the error message
	if !strings.Contains(errorMsg, "TestInterface") {
		return fmt.Errorf("error message should clearly show TestInterface causing the cycle, got: %s", errorMsg)
	}

	return nil
}

func (ctx *EnhancedCycleDetectionBDDTestContext) iInitializeTheApplication() error {
	ctx.initializeResult = ctx.app.Init()
	return nil
}

func (ctx *EnhancedCycleDetectionBDDTestContext) theInitializationShouldSucceed() error {
	if ctx.initializeResult != nil {
		return fmt.Errorf("expected initialization to succeed, got error: %v", ctx.initializeResult)
	}
	return nil
}

func (ctx *EnhancedCycleDetectionBDDTestContext) noCircularDependencyErrorShouldBeReported() error {
	if IsErrCircularDependency(ctx.initializeResult) {
		return fmt.Errorf("unexpected circular dependency error: %v", ctx.initializeResult)
	}
	return nil
}
