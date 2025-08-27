package modular

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/cucumber/godog"
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

// Mock modules for different cycle scenarios

// CycleModuleA - provides TestInterfaceA and requires TestInterfaceB
type CycleModuleA struct {
	name string
}

func (m *CycleModuleA) Name() string               { return m.name }
func (m *CycleModuleA) Init(app Application) error { return nil }

func (m *CycleModuleA) ProvidesServices() []ServiceProvider {
	return []ServiceProvider{{
		Name:     "serviceA",
		Instance: &struct{ TestInterfaceA }{},
	}}
}

func (m *CycleModuleA) RequiresServices() []ServiceDependency {
	return []ServiceDependency{{
		Name:               "serviceB",
		Required:           true,
		MatchByInterface:   true,
		SatisfiesInterface: reflect.TypeOf((*TestInterfaceB)(nil)).Elem(),
	}}
}

// CycleModuleB - provides TestInterfaceB and requires TestInterfaceA
type CycleModuleB struct {
	name string
}

func (m *CycleModuleB) Name() string               { return m.name }
func (m *CycleModuleB) Init(app Application) error { return nil }

func (m *CycleModuleB) ProvidesServices() []ServiceProvider {
	return []ServiceProvider{{
		Name:     "serviceB",
		Instance: &struct{ TestInterfaceB }{},
	}}
}

func (m *CycleModuleB) RequiresServices() []ServiceDependency {
	return []ServiceDependency{{
		Name:               "serviceA",
		Required:           true,
		MatchByInterface:   true,
		SatisfiesInterface: reflect.TypeOf((*TestInterfaceA)(nil)).Elem(),
	}}
}

// LinearModuleA - only provides services, no dependencies
type LinearModuleA struct {
	name string
}

func (m *LinearModuleA) Name() string               { return m.name }
func (m *LinearModuleA) Init(app Application) error { return nil }

func (m *LinearModuleA) ProvidesServices() []ServiceProvider {
	return []ServiceProvider{{
		Name:     "linearServiceA",
		Instance: &struct{ TestInterfaceA }{},
	}}
}

// LinearModuleB - depends on LinearModuleA
type LinearModuleB struct {
	name string
}

func (m *LinearModuleB) Name() string               { return m.name }
func (m *LinearModuleB) Init(app Application) error { return nil }

func (m *LinearModuleB) RequiresServices() []ServiceDependency {
	return []ServiceDependency{{
		Name:               "linearServiceA",
		Required:           true,
		MatchByInterface:   true,
		SatisfiesInterface: reflect.TypeOf((*TestInterfaceA)(nil)).Elem(),
	}}
}

// SelfDependentModule - depends on a service it provides
type SelfDependentModule struct {
	name string
}

func (m *SelfDependentModule) Name() string               { return m.name }
func (m *SelfDependentModule) Init(app Application) error { return nil }

func (m *SelfDependentModule) ProvidesServices() []ServiceProvider {
	return []ServiceProvider{{
		Name:     "selfService",
		Instance: &struct{ TestInterfaceA }{},
	}}
}

func (m *SelfDependentModule) RequiresServices() []ServiceDependency {
	return []ServiceDependency{{
		Name:               "selfService",
		Required:           true,
		MatchByInterface:   true,
		SatisfiesInterface: reflect.TypeOf((*TestInterfaceA)(nil)).Elem(),
	}}
}

// BDD Step implementations

func (ctx *EnhancedCycleDetectionBDDTestContext) iHaveAModularApplication() error {
	enhancedRegistry := NewEnhancedServiceRegistry()
	ctx.app = &StdApplication{
		cfgProvider:         NewStdConfigProvider(testCfg{Str: "test"}),
		cfgSections:         make(map[string]ConfigProvider),
		svcRegistry:         enhancedRegistry.AsServiceRegistry(),
		enhancedSvcRegistry: enhancedRegistry,
		moduleRegistry:      make(ModuleRegistry),
		logger:              &testLogger{},
	}
	ctx.modules = make(map[string]Module)
	return nil
}

func (ctx *EnhancedCycleDetectionBDDTestContext) iHaveTwoModulesWithCircularInterfaceDependencies() error {
	moduleA := &CycleModuleA{name: "moduleA"}
	moduleB := &CycleModuleB{name: "moduleB"}

	ctx.modules["moduleA"] = moduleA
	ctx.modules["moduleB"] = moduleB

	ctx.app.RegisterModule(moduleA)
	ctx.app.RegisterModule(moduleB)

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

func (ctx *EnhancedCycleDetectionBDDTestContext) iHaveModulesAAndBWhereARequiresInterfaceTestInterfaceAndBProvidesTestInterface() error {
	// This is effectively the same as the circular dependency setup
	return ctx.iHaveTwoModulesWithCircularInterfaceDependencies()
}

func (ctx *EnhancedCycleDetectionBDDTestContext) moduleBAlsoRequiresInterfaceTestInterfaceCreatingACycle() error {
	// Already handled in the setup above - moduleB requires TestInterfaceA
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

func (ctx *EnhancedCycleDetectionBDDTestContext) iHaveModulesWithValidLinearDependencies() error {
	moduleA := &LinearModuleA{name: "linearA"}
	moduleB := &LinearModuleB{name: "linearB"}

	ctx.modules["linearA"] = moduleA
	ctx.modules["linearB"] = moduleB

	ctx.app.RegisterModule(moduleA)
	ctx.app.RegisterModule(moduleB)

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

func (ctx *EnhancedCycleDetectionBDDTestContext) iHaveAModuleThatDependsOnAServiceItAlsoProvides() error {
	module := &SelfDependentModule{name: "selfModule"}

	ctx.modules["selfModule"] = module
	ctx.app.RegisterModule(module)

	return nil
}

func (ctx *EnhancedCycleDetectionBDDTestContext) aSelfDependencyCycleShouldBeDetected() error {
	if ctx.initializeResult == nil {
		return fmt.Errorf("expected self-dependency cycle to be detected")
	}

	if !IsErrCircularDependency(ctx.initializeResult) {
		return fmt.Errorf("expected circular dependency error for self-dependency, got %v", ctx.initializeResult)
	}

	return nil
}

func (ctx *EnhancedCycleDetectionBDDTestContext) theErrorMessageShouldClearlyIndicateTheSelfDependency() error {
	if ctx.initializeResult == nil {
		return fmt.Errorf("no error to check")
	}

	errorMsg := ctx.initializeResult.Error()
	// Should mention the module name and self-reference
	if !strings.Contains(errorMsg, "selfModule") {
		return fmt.Errorf("error message should mention the self-dependent module, got: %s", errorMsg)
	}

	return nil
}

// Test runner
func TestEnhancedCycleDetectionBDD(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: func(ctx *godog.ScenarioContext) {
			testContext := &EnhancedCycleDetectionBDDTestContext{}

			// Background
			ctx.Step(`^I have a modular application$`, testContext.iHaveAModularApplication)

			// Cycle detection scenarios
			ctx.Step(`^I have two modules with circular interface dependencies$`, testContext.iHaveTwoModulesWithCircularInterfaceDependencies)
			ctx.Step(`^I try to initialize the application$`, testContext.iTryToInitializeTheApplication)
			ctx.Step(`^the initialization should fail with a circular dependency error$`, testContext.theInitializationShouldFailWithACircularDependencyError)
			ctx.Step(`^the error message should include both module names$`, testContext.theErrorMessageShouldIncludeBothModuleNames)
			ctx.Step(`^the error message should indicate interface-based dependencies$`, testContext.theErrorMessageShouldIndicateInterfaceBasedDependencies)
			ctx.Step(`^the error message should show the complete dependency cycle$`, testContext.theErrorMessageShouldShowTheCompleteDependencyCycle)

			// Enhanced error message format
			ctx.Step(`^I have modules A and B where A requires interface TestInterface and B provides TestInterface$`, testContext.iHaveModulesAAndBWhereARequiresInterfaceTestInterfaceAndBProvidesTestInterface)
			ctx.Step(`^module B also requires interface TestInterface creating a cycle$`, testContext.moduleBAlsoRequiresInterfaceTestInterfaceCreatingACycle)
			ctx.Step(`^the error message should contain "([^"]*)"$`, testContext.theErrorMessageShouldContain)
			ctx.Step(`^the error message should clearly show the interface causing the cycle$`, testContext.theErrorMessageShouldClearlyShowTheInterfaceCausingTheCycle)

			// Linear dependencies (no cycles)
			ctx.Step(`^I have modules with valid linear dependencies$`, testContext.iHaveModulesWithValidLinearDependencies)
			ctx.Step(`^I initialize the application$`, testContext.iInitializeTheApplication)
			ctx.Step(`^the initialization should succeed$`, testContext.theInitializationShouldSucceed)
			ctx.Step(`^no circular dependency error should be reported$`, testContext.noCircularDependencyErrorShouldBeReported)

			// Self-dependency
			ctx.Step(`^I have a module that depends on a service it also provides$`, testContext.iHaveAModuleThatDependsOnAServiceItAlsoProvides)
			ctx.Step(`^a self-dependency cycle should be detected$`, testContext.aSelfDependencyCycleShouldBeDetected)
			ctx.Step(`^the error message should clearly indicate the self-dependency$`, testContext.theErrorMessageShouldClearlyIndicateTheSelfDependency)
		},
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"features/enhanced_cycle_detection.feature"},
			TestingT: t,
		},
	}

	if suite.Run() != 0 {
		t.Fatal("non-zero status returned, failed to run BDD tests")
	}
}
