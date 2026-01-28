package modular

import (
	"fmt"
	"strings"
	"testing"

	"github.com/cucumber/godog"
)

// BDD Step implementations for cycle detection scenarios

func (ctx *EnhancedCycleDetectionBDDTestContext) iHaveTwoModulesWithCircularInterfaceDependencies() error {
	moduleA := &CycleModuleA{name: "moduleA"}
	moduleB := &CycleModuleB{name: "moduleB"}

	ctx.modules["moduleA"] = moduleA
	ctx.modules["moduleB"] = moduleB

	ctx.app.RegisterModule(moduleA)
	ctx.app.RegisterModule(moduleB)

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

func (ctx *EnhancedCycleDetectionBDDTestContext) iHaveModulesWithValidLinearDependencies() error {
	moduleA := &LinearModuleA{name: "linearA"}
	moduleB := &LinearModuleB{name: "linearB"}

	ctx.modules["linearA"] = moduleA
	ctx.modules["linearB"] = moduleB

	ctx.app.RegisterModule(moduleA)
	ctx.app.RegisterModule(moduleB)

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

	// With improved self-interface pruning, a self-required interface dependency
	// manifests as an unsatisfied required service instead of an artificial cycle.
	// Accept either a circular dependency error (legacy behavior) or a required
	// service not found error referencing the self module.
	if !IsErrCircularDependency(ctx.initializeResult) {
		// Fallback acceptance: required service not found for the module's own interface
		if !strings.Contains(ctx.initializeResult.Error(), "required service not found") || !strings.Contains(ctx.initializeResult.Error(), "selfModule") {
			return fmt.Errorf("expected circular dependency or unsatisfied self service error, got %v", ctx.initializeResult)
		}
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

// Missing step implementations for complex scenarios

func (ctx *EnhancedCycleDetectionBDDTestContext) iHaveModulesWithBothNamedServiceDependenciesAndInterfaceDependencies() error {
	moduleA := &MixedDependencyModuleA{name: "mixedA"}
	moduleB := &MixedDependencyModuleB{name: "mixedB"}

	ctx.modules["mixedA"] = moduleA
	ctx.modules["mixedB"] = moduleB

	ctx.app.RegisterModule(moduleA)
	ctx.app.RegisterModule(moduleB)

	return nil
}

func (ctx *EnhancedCycleDetectionBDDTestContext) theDependenciesFormACircularChain() error {
	// Dependencies are already set up in the modules - mixedA requires namedServiceB, mixedB requires interface TestInterfaceA
	return nil
}

func (ctx *EnhancedCycleDetectionBDDTestContext) theErrorMessageShouldDistinguishBetweenInterfaceAndNamedDependencies() error {
	if ctx.initializeResult == nil {
		return fmt.Errorf("no error to check")
	}

	errorMsg := ctx.initializeResult.Error()
	// Should contain both service: and interface: markers
	hasService := strings.Contains(errorMsg, "(service:")
	hasInterface := strings.Contains(errorMsg, "(interface:")

	if !hasService || !hasInterface {
		return fmt.Errorf("error message should distinguish between service and interface dependencies, got: %s", errorMsg)
	}

	return nil
}

func (ctx *EnhancedCycleDetectionBDDTestContext) bothDependencyTypesShouldBeIncludedInTheCycleDescription() error {
	if ctx.initializeResult == nil {
		return fmt.Errorf("no error to check")
	}

	errorMsg := ctx.initializeResult.Error()
	// Should show the complete cycle with both dependency types
	if !strings.Contains(errorMsg, "cycle:") {
		return fmt.Errorf("error message should contain cycle description, got: %s", errorMsg)
	}

	return nil
}

func (ctx *EnhancedCycleDetectionBDDTestContext) iHaveModulesABAndCWhereADependsOnBBDependsOnCAndCDependsOnA() error {
	moduleA := &ComplexCycleModuleA{name: "complexA"}
	moduleB := &ComplexCycleModuleB{name: "complexB"}
	moduleC := &ComplexCycleModuleC{name: "complexC"}

	ctx.modules["complexA"] = moduleA
	ctx.modules["complexB"] = moduleB
	ctx.modules["complexC"] = moduleC

	ctx.app.RegisterModule(moduleA)
	ctx.app.RegisterModule(moduleB)
	ctx.app.RegisterModule(moduleC)

	return nil
}

func (ctx *EnhancedCycleDetectionBDDTestContext) theCompleteCyclePathShouldBeShownInTheErrorMessage() error {
	if ctx.initializeResult == nil {
		return fmt.Errorf("no error to check")
	}

	errorMsg := ctx.initializeResult.Error()
	if !strings.Contains(errorMsg, "cycle:") {
		return fmt.Errorf("error message should show complete cycle path, got: %s", errorMsg)
	}

	// Should contain arrow notation showing the path
	if !strings.Contains(errorMsg, "→") {
		return fmt.Errorf("error message should use arrow notation for cycle path, got: %s", errorMsg)
	}

	return nil
}

func (ctx *EnhancedCycleDetectionBDDTestContext) allThreeModulesShouldBeMentionedInTheCycleDescription() error {
	if ctx.initializeResult == nil {
		return fmt.Errorf("no error to check")
	}

	errorMsg := ctx.initializeResult.Error()
	requiredModules := []string{"complexA", "complexB", "complexC"}

	for _, module := range requiredModules {
		if !strings.Contains(errorMsg, module) {
			return fmt.Errorf("error message should mention all three modules (%v), got: %s", requiredModules, errorMsg)
		}
	}

	return nil
}

func (ctx *EnhancedCycleDetectionBDDTestContext) iHaveMultipleInterfacesWithSimilarNamesCausingCycles() error {
	moduleA := &DisambiguationModuleA{name: "disambigA"}
	moduleB := &DisambiguationModuleB{name: "disambigB"}

	ctx.modules["disambigA"] = moduleA
	ctx.modules["disambigB"] = moduleB

	ctx.app.RegisterModule(moduleA)
	ctx.app.RegisterModule(moduleB)

	return nil
}

func (ctx *EnhancedCycleDetectionBDDTestContext) cycleDetectionRuns() error {
	ctx.initializeResult = ctx.app.Init()
	return nil
}

func (ctx *EnhancedCycleDetectionBDDTestContext) interfaceNamesInErrorMessagesShouldBeFullyQualified() error {
	if ctx.initializeResult == nil {
		return fmt.Errorf("no error to check")
	}

	errorMsg := ctx.initializeResult.Error()
	// Should contain fully qualified interface names to avoid ambiguity
	// Look for package prefix in interface names
	if !strings.Contains(errorMsg, "modular.EnhancedTestInterface") && !strings.Contains(errorMsg, "modular.AnotherEnhancedTestInterface") {
		return fmt.Errorf("error message should contain fully qualified interface names, got: %s", errorMsg)
	}

	return nil
}

func (ctx *EnhancedCycleDetectionBDDTestContext) thereShouldBeNoAmbiguityAboutWhichInterfaceCausedTheCycle() error {
	if ctx.initializeResult == nil {
		return fmt.Errorf("no error to check")
	}

	errorMsg := ctx.initializeResult.Error()
	// The interface names should be clearly distinguishable
	if strings.Contains(errorMsg, "EnhancedTestInterface") && strings.Contains(errorMsg, "AnotherEnhancedTestInterface") {
		// Both interfaces mentioned - good disambiguation
		return nil
	}

	return fmt.Errorf("error message should clearly distinguish between different interfaces, got: %s", errorMsg)
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

			// Mixed dependency types
			ctx.Step(`^I have modules with both named service dependencies and interface dependencies$`, testContext.iHaveModulesWithBothNamedServiceDependenciesAndInterfaceDependencies)
			ctx.Step(`^the dependencies form a circular chain$`, testContext.theDependenciesFormACircularChain)
			ctx.Step(`^the error message should distinguish between interface and named dependencies$`, testContext.theErrorMessageShouldDistinguishBetweenInterfaceAndNamedDependencies)
			ctx.Step(`^both dependency types should be included in the cycle description$`, testContext.bothDependencyTypesShouldBeIncludedInTheCycleDescription)

			// Complex multi-module cycles
			ctx.Step(`^I have modules A, B, and C where A depends on B, B depends on C, and C depends on A$`, testContext.iHaveModulesABAndCWhereADependsOnBBDependsOnCAndCDependsOnA)
			ctx.Step(`^the complete cycle path should be shown in the error message$`, testContext.theCompleteCyclePathShouldBeShownInTheErrorMessage)
			ctx.Step(`^all three modules should be mentioned in the cycle description$`, testContext.allThreeModulesShouldBeMentionedInTheCycleDescription)

			// Interface name disambiguation
			ctx.Step(`^I have multiple interfaces with similar names causing cycles$`, testContext.iHaveMultipleInterfacesWithSimilarNamesCausingCycles)
			ctx.Step(`^cycle detection runs$`, testContext.cycleDetectionRuns)
			ctx.Step(`^interface names in error messages should be fully qualified$`, testContext.interfaceNamesInErrorMessagesShouldBeFullyQualified)
			ctx.Step(`^there should be no ambiguity about which interface caused the cycle$`, testContext.thereShouldBeNoAmbiguityAboutWhichInterfaceCausedTheCycle)
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
