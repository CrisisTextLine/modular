package modular

import (
	"fmt"
)

// BDD Test Context for Enhanced Service Registry
type EnhancedServiceRegistryBDDContext struct {
	app                Application
	modules            map[string]Module
	services           map[string]any
	lastError          error
	retrievedServices  []*ServiceRegistryEntry
	servicesByModule   []string
	serviceEntry       *ServiceRegistryEntry
	serviceEntryExists bool
}

// Test interface for interface-based discovery tests
type TestServiceInterface interface {
	DoSomething() string
}

// Mock implementation of TestServiceInterface
type EnhancedMockTestService struct {
	identifier string
}

func (m *EnhancedMockTestService) DoSomething() string {
	return fmt.Sprintf("Service: %s", m.identifier)
}

// BDD Step implementations

func (ctx *EnhancedServiceRegistryBDDContext) iHaveAModularApplicationWithEnhancedServiceRegistry() error {
	// Use the builder pattern for cleaner application creation
	app, err := NewApplication(
		WithLogger(&testLogger{}),
		WithConfigProvider(NewStdConfigProvider(testCfg{Str: "test"})),
	)
	if err != nil {
		return err
	}
	ctx.app = app
	ctx.modules = make(map[string]Module)
	ctx.services = make(map[string]any)
	return nil
}

func (ctx *EnhancedServiceRegistryBDDContext) iRegisterTheModuleAndInitializeTheApplication() error {
	err := ctx.app.Init()
	ctx.lastError = err
	return err
}

func (ctx *EnhancedServiceRegistryBDDContext) theServiceShouldBeRegisteredWithModuleAssociation() error {
	// Check that services exist in the registry
	for serviceName := range ctx.services {
		var service *EnhancedMockTestService
		err := ctx.app.GetService(serviceName, &service)
		if err != nil {
			return fmt.Errorf("service %s not found: %w", serviceName, err)
		}
	}
	return nil
}

func (ctx *EnhancedServiceRegistryBDDContext) iShouldBeAbleToRetrieveTheServiceEntryWithModuleInformation() error {
	for serviceName := range ctx.services {
		entry, exists := ctx.app.GetServiceEntry(serviceName)
		if !exists {
			return fmt.Errorf("service entry for %s not found", serviceName)
		}

		if entry.OriginalName != serviceName {
			return fmt.Errorf("expected original name %s, got %s", serviceName, entry.OriginalName)
		}

		if entry.ModuleName == "" {
			return fmt.Errorf("module name should not be empty for service %s", serviceName)
		}
	}
	return nil
}

func (ctx *EnhancedServiceRegistryBDDContext) theApplicationInitializes() error {
	err := ctx.app.Init()
	ctx.lastError = err
	return err
}

func (ctx *EnhancedServiceRegistryBDDContext) iHaveAServiceRegisteredByModule(serviceName, moduleName string) error {
	service := &EnhancedMockTestService{identifier: serviceName}
	module := &SingleServiceModule{
		name:        moduleName,
		serviceName: serviceName,
		service:     service,
	}

	ctx.modules[moduleName] = module
	ctx.services[serviceName] = service
	ctx.app.RegisterModule(module)

	// Initialize to register the service
	err := ctx.app.Init()
	ctx.lastError = err
	return err
}

func (ctx *EnhancedServiceRegistryBDDContext) iRetrieveTheServiceEntryByName() error {
	// Use the last registered service name
	var serviceName string
	for name := range ctx.services {
		serviceName = name
		break // Use the first service
	}

	entry, exists := ctx.app.GetServiceEntry(serviceName)
	ctx.serviceEntry = entry
	ctx.serviceEntryExists = exists
	return nil
}

func (ctx *EnhancedServiceRegistryBDDContext) theEntryShouldContainTheOriginalNameActualNameModuleNameAndModuleType() error {
	if !ctx.serviceEntryExists {
		return fmt.Errorf("service entry does not exist")
	}

	if ctx.serviceEntry.OriginalName == "" {
		return fmt.Errorf("original name is empty")
	}
	if ctx.serviceEntry.ActualName == "" {
		return fmt.Errorf("actual name is empty")
	}
	if ctx.serviceEntry.ModuleName == "" {
		return fmt.Errorf("module name is empty")
	}
	if ctx.serviceEntry.ModuleType == nil {
		return fmt.Errorf("module type is nil")
	}

	return nil
}

func (ctx *EnhancedServiceRegistryBDDContext) iShouldBeAbleToAccessTheActualServiceInstance() error {
	if !ctx.serviceEntryExists {
		return fmt.Errorf("service entry does not exist")
	}

	// Try to cast to expected type
	if _, ok := ctx.serviceEntry.Service.(*EnhancedMockTestService); !ok {
		return fmt.Errorf("service instance is not of expected type")
	}

	return nil
}

func (ctx *EnhancedServiceRegistryBDDContext) iHaveServicesRegisteredThroughBothOldAndNewPatterns() error {
	// Register through old pattern (direct registry access)
	oldService := &EnhancedMockTestService{identifier: "oldPattern"}
	err := ctx.app.RegisterService("oldService", oldService)
	if err != nil {
		return err
	}

	// Register through new pattern (module-based)
	return ctx.iHaveAServiceRegisteredByModule("newService", "NewModule")
}

func (ctx *EnhancedServiceRegistryBDDContext) iAccessServicesThroughTheBackwardsCompatibleInterface() error {
	var oldService, newService EnhancedMockTestService

	errOld := ctx.app.GetService("oldService", &oldService)
	errNew := ctx.app.GetService("newService", &newService)

	if errOld != nil || errNew != nil {
		return fmt.Errorf("not all services accessible: old=%v, new=%v", errOld, errNew)
	}

	return nil
}

func (ctx *EnhancedServiceRegistryBDDContext) allServicesShouldBeAccessibleRegardlessOfRegistrationMethod() error {
	return ctx.iAccessServicesThroughTheBackwardsCompatibleInterface()
}

func (ctx *EnhancedServiceRegistryBDDContext) theServiceRegistryMapShouldContainAllServices() error {
	registry := ctx.app.SvcRegistry()

	if _, exists := registry["oldService"]; !exists {
		return fmt.Errorf("old service not found in registry map")
	}
	if _, exists := registry["newService"]; !exists {
		return fmt.Errorf("new service not found in registry map")
	}

	return nil
}
