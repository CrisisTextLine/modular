package modular

import (
	"context"
	"testing"
)

// This file demonstrates the correct way to depend on a service during module initialization.
// It shows that using RequiresServices() with Required:true ensures proper initialization order.

// mockSchedulerModule simulates the scheduler module
type mockSchedulerModule struct {
	testModule
	name           string
	registeredJobs []string
}

func (m *mockSchedulerModule) Name() string {
	return m.name
}

func (m *mockSchedulerModule) Init(app Application) error {
	// Scheduler initializes its internal state
	m.registeredJobs = make([]string, 0)
	return nil
}

func (m *mockSchedulerModule) ProvidesServices() []ServiceProvider {
	return []ServiceProvider{
		{
			Name:        "scheduler.provider",
			Description: "Job scheduling service",
			Instance:    m,
		},
	}
}

func (m *mockSchedulerModule) RegisterJob(name string) {
	m.registeredJobs = append(m.registeredJobs, name)
}

func (m *mockSchedulerModule) Start(ctx context.Context) error {
	return nil
}

// mockJobsModule simulates a module that depends on the scheduler
type mockJobsModule struct {
	testModule
	name           string
	jobsRegistered bool
	scheduler      *mockSchedulerModule
}

func (m *mockJobsModule) Name() string {
	return m.name
}

func (m *mockJobsModule) Init(app Application) error {
	// Get scheduler service during Init
	// This works because RequiresServices() ensures scheduler is initialized first
	var scheduler *mockSchedulerModule
	err := app.GetService("scheduler.provider", &scheduler)
	if err != nil {
		return err
	}

	m.scheduler = scheduler

	// Register jobs with scheduler
	scheduler.RegisterJob("daily-cleanup")
	scheduler.RegisterJob("hourly-report")
	scheduler.RegisterJob("weekly-backup")

	m.jobsRegistered = true
	return nil
}

// RequiresServices declares the dependency on scheduler service
// This is REQUIRED to ensure proper initialization order
func (m *mockJobsModule) RequiresServices() []ServiceDependency {
	return []ServiceDependency{
		{
			Name:     "scheduler.provider",
			Required: true,
		},
	}
}

func (m *mockJobsModule) ProvidesServices() []ServiceProvider {
	return nil
}

// TestSchedulerDependencyPattern validates the correct dependency pattern
func TestSchedulerDependencyPattern(t *testing.T) {
	app := NewStdApplication(NewStdConfigProvider(testCfg{Str: "test"}), &logger{t})

	schedulerModule := &mockSchedulerModule{name: "scheduler"}
	jobsModule := &mockJobsModule{name: "jobs"}

	// Register in reverse alphabetical order to test dependency resolution
	app.RegisterModule(jobsModule)
	app.RegisterModule(schedulerModule)

	err := app.Init()
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Verify initialization order was correct
	if !jobsModule.jobsRegistered {
		t.Error("Jobs were not registered - likely initialization order issue")
	}

	if len(schedulerModule.registeredJobs) != 3 {
		t.Errorf("Expected 3 registered jobs, got %d", len(schedulerModule.registeredJobs))
	}
}
