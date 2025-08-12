package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/CrisisTextLine/modular"
	"github.com/CrisisTextLine/modular/modules/eventbus"
)

// testLogger is a simple logger for the example
type testLogger struct{}

func (l *testLogger) Debug(msg string, args ...interface{}) {
	// Skip debug messages for cleaner output
}

func (l *testLogger) Info(msg string, args ...interface{}) {
	// Skip info messages for cleaner output
}

func (l *testLogger) Warn(msg string, args ...interface{}) {
	fmt.Printf("WARN: %s %v\n", msg, args)
}

func (l *testLogger) Error(msg string, args ...interface{}) {
	fmt.Printf("ERROR: %s %v\n", msg, args)
}

// AppConfig defines the main application configuration
type AppConfig struct {
	Name        string `yaml:"name" desc:"Application name"`
	Environment string `yaml:"environment" desc:"Environment (dev, staging, prod)"`
}

// UserEvent represents a user-related event
type UserEvent struct {
	UserID    string    `json:"userId"`
	Action    string    `json:"action"`
	Timestamp time.Time `json:"timestamp"`
}

// AnalyticsEvent represents an analytics event
type AnalyticsEvent struct {
	SessionID string    `json:"sessionId"`
	EventType string    `json:"eventType"`
	Page      string    `json:"page"`
	Timestamp time.Time `json:"timestamp"`
}

// SystemEvent represents a system event
type SystemEvent struct {
	Component string    `json:"component"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

func main() {
	ctx := context.Background()

	// Create application configuration
	appConfig := &AppConfig{
		Name:        "Multi-Engine EventBus Demo",
		Environment: "development",
	}

	// Create eventbus configuration with multiple engines and routing
	eventbusConfig := &eventbus.EventBusConfig{
		Engines: []eventbus.EngineConfig{
			{
				Name: "memory-fast",
				Type: "memory",
				Config: map[string]interface{}{
					"maxEventQueueSize":      500,
					"defaultEventBufferSize": 10,
					"workerCount":            3,
					"retentionDays":          1,
				},
			},
			{
				Name: "memory-reliable",
				Type: "custom",
				Config: map[string]interface{}{
					"enableMetrics":          true,
					"maxEventQueueSize":      2000,
					"defaultEventBufferSize": 50,
					"metricsInterval":        "30s",
				},
			},
		},
		Routing: []eventbus.RoutingRule{
			{
				Topics: []string{"user.*", "auth.*"},
				Engine: "memory-fast",
			},
			{
				Topics: []string{"analytics.*", "metrics.*"},
				Engine: "memory-reliable",
			},
			{
				Topics: []string{"*"}, // Fallback for all other topics
				Engine: "memory-reliable",
			},
		},
	}

	// Initialize application
	mainConfigProvider := modular.NewStdConfigProvider(appConfig)
	app := modular.NewStdApplication(mainConfigProvider, &testLogger{})

	// Register configurations
	app.RegisterConfigSection("eventbus", modular.NewStdConfigProvider(eventbusConfig))

	// Register modules
	app.RegisterModule(eventbus.NewModule())

	// Initialize application
	err := app.Init()
	if err != nil {
		log.Fatal("Failed to initialize application:", err)
	}

	// Get services
	var eventBusService *eventbus.EventBusModule
	err = app.GetService("eventbus.provider", &eventBusService)
	if err != nil {
		log.Fatal("Failed to get eventbus service:", err)
	}

	// Start application
	err = app.Start()
	if err != nil {
		log.Fatal("Failed to start application:", err)
	}

	fmt.Printf("🚀 Started %s in %s environment\n", appConfig.Name, appConfig.Environment)
	fmt.Println("📊 Multi-Engine EventBus Configuration:")
	fmt.Println("  - memory-fast: Handles user.* and auth.* topics")
	fmt.Println("  - memory-reliable: Handles analytics.*, metrics.*, and fallback topics")
	fmt.Println()

	// Set up event handlers
	setupEventHandlers(ctx, eventBusService)

	// Demonstrate multi-engine event publishing
	demonstrateMultiEngineEvents(ctx, eventBusService)

	// Wait a bit for event processing
	fmt.Println("⏳ Processing events...")
	time.Sleep(2 * time.Second)

	// Show routing information
	showRoutingInfo(eventBusService)

	// Graceful shutdown
	fmt.Println("\n🛑 Shutting down...")
	err = app.Stop()
	if err != nil {
		log.Printf("Error during shutdown: %v", err)
		os.Exit(1)
	}

	fmt.Println("✅ Application shutdown complete")
}

func setupEventHandlers(ctx context.Context, eventBus *eventbus.EventBusModule) {
	// User event handlers (routed to memory-fast engine)
	eventBus.Subscribe(ctx, "user.registered", func(ctx context.Context, event eventbus.Event) error {
		userEvent := event.Payload.(UserEvent)
		fmt.Printf("🔵 [MEMORY-FAST] User registered: %s (action: %s)\n", 
			userEvent.UserID, userEvent.Action)
		return nil
	})

	eventBus.Subscribe(ctx, "user.login", func(ctx context.Context, event eventbus.Event) error {
		userEvent := event.Payload.(UserEvent)
		fmt.Printf("🔵 [MEMORY-FAST] User login: %s at %s\n", 
			userEvent.UserID, userEvent.Timestamp.Format("15:04:05"))
		return nil
	})

	eventBus.Subscribe(ctx, "auth.failed", func(ctx context.Context, event eventbus.Event) error {
		userEvent := event.Payload.(UserEvent)
		fmt.Printf("🔴 [MEMORY-FAST] Auth failed for user: %s\n", userEvent.UserID)
		return nil
	})

	// Analytics event handlers (routed to memory-reliable engine)
	eventBus.SubscribeAsync(ctx, "analytics.pageview", func(ctx context.Context, event eventbus.Event) error {
		analyticsEvent := event.Payload.(AnalyticsEvent)
		fmt.Printf("📈 [MEMORY-RELIABLE] Page view: %s (session: %s)\n", 
			analyticsEvent.Page, analyticsEvent.SessionID)
		return nil
	})

	eventBus.SubscribeAsync(ctx, "analytics.click", func(ctx context.Context, event eventbus.Event) error {
		analyticsEvent := event.Payload.(AnalyticsEvent)
		fmt.Printf("📈 [MEMORY-RELIABLE] Click event: %s on %s\n", 
			analyticsEvent.EventType, analyticsEvent.Page)
		return nil
	})

	// System event handlers (fallback routing to memory-reliable engine)
	eventBus.Subscribe(ctx, "system.health", func(ctx context.Context, event eventbus.Event) error {
		systemEvent := event.Payload.(SystemEvent)
		fmt.Printf("⚙️  [MEMORY-RELIABLE] System %s: %s - %s\n", 
			systemEvent.Level, systemEvent.Component, systemEvent.Message)
		return nil
	})
}

func demonstrateMultiEngineEvents(ctx context.Context, eventBus *eventbus.EventBusModule) {
	fmt.Println("🎯 Publishing events to different engines based on topic routing:")
	fmt.Println()

	now := time.Now()

	// User events (routed to memory-fast engine)
	userEvents := []UserEvent{
		{UserID: "user123", Action: "register", Timestamp: now},
		{UserID: "user456", Action: "login", Timestamp: now.Add(1 * time.Second)},
		{UserID: "user789", Action: "failed_login", Timestamp: now.Add(2 * time.Second)},
	}

	for i, event := range userEvents {
		var topic string
		switch event.Action {
		case "register":
			topic = "user.registered"
		case "login":
			topic = "user.login"
		case "failed_login":
			topic = "auth.failed"
		}

		err := eventBus.Publish(ctx, topic, event)
		if err != nil {
			fmt.Printf("Error publishing user event: %v\n", err)
		}

		if i < len(userEvents)-1 {
			time.Sleep(200 * time.Millisecond)
		}
	}

	time.Sleep(500 * time.Millisecond)

	// Analytics events (routed to memory-reliable engine)
	analyticsEvents := []AnalyticsEvent{
		{SessionID: "sess123", EventType: "pageview", Page: "/dashboard", Timestamp: now},
		{SessionID: "sess123", EventType: "click", Page: "/dashboard", Timestamp: now.Add(1 * time.Second)},
		{SessionID: "sess456", EventType: "pageview", Page: "/profile", Timestamp: now.Add(2 * time.Second)},
	}

	for i, event := range analyticsEvents {
		var topic string
		switch event.EventType {
		case "pageview":
			topic = "analytics.pageview"
		case "click":
			topic = "analytics.click"
		}

		err := eventBus.Publish(ctx, topic, event)
		if err != nil {
			fmt.Printf("Error publishing analytics event: %v\n", err)
		}

		if i < len(analyticsEvents)-1 {
			time.Sleep(200 * time.Millisecond)
		}
	}

	time.Sleep(500 * time.Millisecond)

	// System events (fallback routing to memory-reliable engine)
	systemEvents := []SystemEvent{
		{Component: "database", Level: "info", Message: "Connection established", Timestamp: now},
		{Component: "cache", Level: "warning", Message: "High memory usage", Timestamp: now.Add(1 * time.Second)},
	}

	for i, event := range systemEvents {
		err := eventBus.Publish(ctx, "system.health", event)
		if err != nil {
			fmt.Printf("Error publishing system event: %v\n", err)
		}

		if i < len(systemEvents)-1 {
			time.Sleep(200 * time.Millisecond)
		}
	}
}

func showRoutingInfo(eventBus *eventbus.EventBusModule) {
	fmt.Println()
	fmt.Println("📋 Event Bus Routing Information:")
	
	// Show how different topics are routed
	topics := []string{
		"user.registered", "user.login", "auth.failed",
		"analytics.pageview", "analytics.click", 
		"system.health", "random.topic",
	}

	if eventBus != nil && eventBus.GetRouter() != nil {
		for _, topic := range topics {
			engine := eventBus.GetRouter().GetEngineForTopic(topic)
			fmt.Printf("  %s -> %s\n", topic, engine)
		}
	}

	// Show active topics and subscriber counts
	activeTopics := eventBus.Topics()
	if len(activeTopics) > 0 {
		fmt.Println()
		fmt.Println("📊 Active Topics and Subscriber Counts:")
		for _, topic := range activeTopics {
			count := eventBus.SubscriberCount(topic)
			fmt.Printf("  %s: %d subscribers\n", topic, count)
		}
	}
}