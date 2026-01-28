package modular

import (
	"fmt"
)

// Test modules for BDD scenarios

// SingleServiceModule provides one service
type SingleServiceModule struct {
	name        string
	serviceName string
	service     any
}

func (m *SingleServiceModule) Name() string               { return m.name }
func (m *SingleServiceModule) Init(app Application) error { return nil }

// Explicitly implement ServiceAware interface
func (m *SingleServiceModule) ProvidesServices() []ServiceProvider {
	return []ServiceProvider{{
		Name:     m.serviceName,
		Instance: m.service,
	}}
}

func (m *SingleServiceModule) RequiresServices() []ServiceDependency {
	return nil // No dependencies for test modules
}

// Ensure the struct implements ServiceAware
var _ ServiceAware = (*SingleServiceModule)(nil)

// ConflictingServiceModule provides a service that might conflict with others
type ConflictingServiceModule struct {
	name        string
	serviceName string
	service     any
}

func (m *ConflictingServiceModule) Name() string               { return m.name }
func (m *ConflictingServiceModule) Init(app Application) error { return nil }

// Explicitly implement ServiceAware interface
func (m *ConflictingServiceModule) ProvidesServices() []ServiceProvider {
	return []ServiceProvider{{
		Name:     m.serviceName,
		Instance: m.service,
	}}
}

func (m *ConflictingServiceModule) RequiresServices() []ServiceDependency {
	return nil // No dependencies for test modules
}

// Ensure the struct implements ServiceAware
var _ ServiceAware = (*ConflictingServiceModule)(nil)

// MultiServiceModule provides multiple services
type MultiServiceModule struct {
	name     string
	services []ServiceProvider
}

func (m *MultiServiceModule) Name() string               { return m.name }
func (m *MultiServiceModule) Init(app Application) error { return nil }

// Explicitly implement ServiceAware interface
func (m *MultiServiceModule) ProvidesServices() []ServiceProvider {
	return m.services
}

func (m *MultiServiceModule) RequiresServices() []ServiceDependency {
	return nil // No dependencies for test modules
}

// Ensure the struct implements ServiceAware
var _ ServiceAware = (*MultiServiceModule)(nil)

// BDD Step implementations for modules

func (ctx *EnhancedServiceRegistryBDDContext) iHaveAModuleThatProvidesAService(moduleName, serviceName string) error {
	service := &EnhancedMockTestService{identifier: fmt.Sprintf("%s:%s", moduleName, serviceName)}
	module := &SingleServiceModule{
		name:        moduleName,
		serviceName: serviceName,
		service:     service,
	}

	ctx.modules[moduleName] = module
	ctx.services[serviceName] = service
	ctx.app.RegisterModule(module)
	return nil
}

func (ctx *EnhancedServiceRegistryBDDContext) iHaveModulesProvidingDifferentServices(moduleA, moduleB, moduleC string) error {
	modules := []struct {
		name    string
		service string
	}{
		{moduleA, "serviceA"},
		{moduleB, "serviceB"},
		{moduleB, "serviceBExtra"}, // ModuleB provides 2 services
		{moduleC, "serviceC"},
	}

	for _, m := range modules {
		service := &EnhancedMockTestService{identifier: m.service}

		// Check if module already exists
		if existingModule, exists := ctx.modules[m.name]; exists {
			// Add to existing multi-service module
			if multiModule, ok := existingModule.(*MultiServiceModule); ok {
				multiModule.services = append(multiModule.services, ServiceProvider{
					Name:     m.service,
					Instance: service,
				})
			}
		} else {
			// Create new module
			module := &MultiServiceModule{
				name: m.name,
				services: []ServiceProvider{{
					Name:     m.service,
					Instance: service,
				}},
			}
			ctx.modules[m.name] = module
			ctx.app.RegisterModule(module)
		}
	}
	return nil
}

func (ctx *EnhancedServiceRegistryBDDContext) iHaveMultipleModulesProvidingServicesThatImplement(interfaceName string) error {
	// Create modules that provide services implementing TestServiceInterface
	for i, moduleName := range []string{"InterfaceModuleA", "InterfaceModuleB", "InterfaceModuleC"} {
		service := &EnhancedMockTestService{identifier: fmt.Sprintf("service%d", i+1)}
		module := &SingleServiceModule{
			name:        moduleName,
			serviceName: fmt.Sprintf("interfaceService%d", i+1),
			service:     service,
		}

		ctx.modules[moduleName] = module
		ctx.app.RegisterModule(module)
	}
	return nil
}

func (ctx *EnhancedServiceRegistryBDDContext) iHaveThreeModulesProvidingServicesImplementingTheSameInterface() error {
	for i, moduleName := range []string{"ConflictModuleA", "ConflictModuleB", "ConflictModuleC"} {
		service := &EnhancedMockTestService{identifier: fmt.Sprintf("conflict%d", i+1)}
		module := &ConflictingServiceModule{
			name:        moduleName,
			serviceName: "conflictService", // Same name for all
			service:     service,
		}

		ctx.modules[moduleName] = module
		ctx.app.RegisterModule(module)
	}
	return nil
}

func (ctx *EnhancedServiceRegistryBDDContext) iHaveAModuleThatProvidesMultipleServicesWithPotentialNameConflicts() error {
	services := []ServiceProvider{
		{Name: "commonService", Instance: &EnhancedMockTestService{identifier: "service1"}},
		{Name: "commonService.extra", Instance: &EnhancedMockTestService{identifier: "service2"}},
		{Name: "commonService", Instance: &EnhancedMockTestService{identifier: "service3"}}, // Conflict with first
	}

	module := &MultiServiceModule{
		name:     "ConflictingModule",
		services: services,
	}

	ctx.modules["ConflictingModule"] = module
	ctx.app.RegisterModule(module)
	return nil
}
