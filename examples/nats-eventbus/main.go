package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
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

// isShuttingDown checks if an error indicates the system is shutting down
func isShuttingDown(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "event bus not started")
}

// AppConfig defines the main application configuration
type AppConfig struct {
	Name        string `yaml:"name" desc:"Application name"`
	Environment string `yaml:"environment" desc:"Environment (dev, staging, prod)"`
}

// OrderEvent represents an order-related event
type OrderEvent struct {
	OrderID   string    `json:"orderId"`
	Action    string    `json:"action"`
	Amount    float64   `json:"amount"`
	Timestamp time.Time `json:"timestamp"`
}

// NotificationEvent represents a notification event
type NotificationEvent struct {
	Type      string    `json:"type"`
	Message   string    `json:"message"`
	Recipient string    `json:"recipient"`
	Timestamp time.Time `json:"timestamp"`
}

// AnalyticsEvent represents an analytics event
type AnalyticsEvent struct {
	EventType string                 `json:"eventType"`
	Data      map[string]interface{} `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
}

// EventTracker tracks published and consumed events for validation
type EventTracker struct {
	mu                 sync.Mutex
	publishedOrders    int
	publishedAnalytics int
	publishedNotifs    int
	consumedOrders     int
	consumedAnalytics  int
	consumedNotifs     int
}

func (et *EventTracker) PublishedOrder() {
	et.mu.Lock()
	defer et.mu.Unlock()
	et.publishedOrders++
}

func (et *EventTracker) PublishedAnalytics() {
	et.mu.Lock()
	defer et.mu.Unlock()
	et.publishedAnalytics++
}

func (et *EventTracker) PublishedNotif() {
	et.mu.Lock()
	defer et.mu.Unlock()
	et.publishedNotifs++
}

func (et *EventTracker) ConsumedOrder() {
	et.mu.Lock()
	defer et.mu.Unlock()
	et.consumedOrders++
}

func (et *EventTracker) ConsumedAnalytics() {
	et.mu.Lock()
	defer et.mu.Unlock()
	et.consumedAnalytics++
}

func (et *EventTracker) ConsumedNotif() {
	et.mu.Lock()
	defer et.mu.Unlock()
	et.consumedNotifs++
}

func (et *EventTracker) GetStats() (pubOrders, pubAnalytics, pubNotifs, consOrders, consAnalytics, consNotifs int) {
	et.mu.Lock()
	defer et.mu.Unlock()
	return et.publishedOrders, et.publishedAnalytics, et.publishedNotifs, et.consumedOrders, et.consumedAnalytics, et.consumedNotifs
}

func (et *EventTracker) Validate() bool {
	et.mu.Lock()
	defer et.mu.Unlock()
	return et.publishedOrders == et.consumedOrders &&
		et.publishedAnalytics == et.consumedAnalytics &&
		et.publishedNotifs == et.consumedNotifs
}

func main() {
	ctx := context.Background()

	// Create application configuration
	appConfig := &AppConfig{
		Name:        "NATS EventBus Demo",
		Environment: "development",
	}

	// Create eventbus configuration with NATS engine
	eventbusConfig := &eventbus.EventBusConfig{
		Engines: []eventbus.EngineConfig{
			{
				Name: "nats-primary",
				Type: "nats",
				Config: map[string]interface{}{
					"url":              "nats://localhost:4222",
					"connectionName":   "nats-eventbus-demo",
					"maxReconnects":    10,
					"reconnectWait":    2,
					"allowReconnect":   true,
					"pingInterval":     20,
					"maxPingsOut":      2,
					"subscribeTimeout": 5,
				},
			},
		},
		Routing: []eventbus.RoutingRule{
			{
				Topics: []string{"*"}, // All topics go to NATS
				Engine: "nats-primary",
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
	fmt.Println("📊 NATS EventBus Configuration:")
	fmt.Println("  - NATS server: localhost:4222")
	fmt.Println("  - All topics routed through NATS")
	fmt.Println()

	// Check if NATS service is available
	checkNATSAvailability()

	// Give the eventbus a moment to fully initialize connections
	time.Sleep(500 * time.Millisecond)

	// Create event tracker for validation
	tracker := &EventTracker{}

	// Set up a wait group for services
	var wg sync.WaitGroup

	// Set up signal handling for graceful shutdown
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	// Start Publisher Service (Service 1)
	wg.Add(1)
	go func() {
		defer wg.Done()
		runPublisherService(ctx, eventBusService, signalChan, tracker)
	}()

	// Start Subscriber Services (Service 2)
	wg.Add(1)
	go func() {
		defer wg.Done()
		runSubscriberService(ctx, eventBusService, signalChan, tracker)
	}()

	// Wait for signal or service completion
	fmt.Println("🔄 Services are running. Press Ctrl+C to stop...")
	fmt.Println()

	// Wait for shutdown signal
	<-signalChan

	// Graceful shutdown
	fmt.Println("\n🛑 Shutting down services...")

	// Wait for services to complete (they will stop when they receive the signal)
	wg.Wait()

	// Wait a moment for async processing to complete
	fmt.Println("⏳ Waiting for event processing to complete...")
	time.Sleep(2 * time.Second)

	// Stop application after services have stopped
	err = app.Stop()
	if err != nil {
		log.Printf("Warning during shutdown: %v", err)
	}

	// Validate event correlation
	pubOrders, pubAnalytics, pubNotifs, consOrders, consAnalytics, consNotifs := tracker.GetStats()

	fmt.Println("\n📊 Event Correlation Report:")
	fmt.Printf("  Orders:      Published: %d, Consumed: %d ✓\n", pubOrders, consOrders)
	fmt.Printf("  Analytics:   Published: %d, Consumed: %d ✓\n", pubAnalytics, consAnalytics)
	fmt.Printf("  Notifications: Published: %d, Consumed: %d ✓\n", pubNotifs, consNotifs)

	if tracker.Validate() {
		fmt.Println("\n✅ Validation PASSED: All published events were consumed")
	} else {
		fmt.Println("\n❌ Validation FAILED: Mismatch between published and consumed events")
		os.Exit(1)
	}

	fmt.Println("✅ Application shutdown complete")
}

// runPublisherService simulates a service that publishes events
func runPublisherService(ctx context.Context, eventBus *eventbus.EventBusModule, stopChan <-chan os.Signal, tracker *EventTracker) {
	fmt.Println("📤 Publisher Service started")

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	orderCounter := 1

	for {
		select {
		case <-stopChan:
			fmt.Println("📤 Publisher Service stopping...")
			return
		case <-ticker.C:
			// Publish order event
			orderEvent := OrderEvent{
				OrderID:   fmt.Sprintf("ORDER-%d", orderCounter),
				Action:    "created",
				Amount:    99.99 + float64(orderCounter),
				Timestamp: time.Now(),
			}

			fmt.Printf("📤 [PUBLISHED] order.created: %s (amount: $%.2f)\n", orderEvent.OrderID, orderEvent.Amount)
			err := eventBus.Publish(ctx, "order.created", orderEvent)
			if err != nil {
				// Errors during shutdown are expected, don't print them
				if !isShuttingDown(err) {
					fmt.Printf("Error publishing order event: %v\n", err)
				}
			} else {
				tracker.PublishedOrder()
			}

			// Publish analytics event
			analyticsEvent := AnalyticsEvent{
				EventType: "order_created",
				Data: map[string]interface{}{
					"order_id": orderEvent.OrderID,
					"amount":   orderEvent.Amount,
				},
				Timestamp: time.Now(),
			}

			fmt.Printf("📤 [PUBLISHED] analytics.order: %s\n", orderEvent.OrderID)
			err = eventBus.Publish(ctx, "analytics.order", analyticsEvent)
			if err != nil {
				// Errors during shutdown are expected, don't print them
				if !isShuttingDown(err) {
					fmt.Printf("Error publishing analytics event: %v\n", err)
				}
			} else {
				tracker.PublishedAnalytics()
			}

			orderCounter++

			// Publish notification every 2 orders
			if orderCounter%2 == 0 {
				notifEvent := NotificationEvent{
					Type:      "order_milestone",
					Message:   fmt.Sprintf("Processed %d orders", orderCounter-1),
					Recipient: "admin@example.com",
					Timestamp: time.Now(),
				}

				fmt.Printf("📤 [PUBLISHED] notification.system: %s\n", notifEvent.Message)
				err = eventBus.Publish(ctx, "notification.system", notifEvent)
				if err != nil {
					// Errors during shutdown are expected, don't print them
					if !isShuttingDown(err) {
						fmt.Printf("Error publishing notification event: %v\n", err)
					}
				} else {
					tracker.PublishedNotif()
				}
			}

			fmt.Println()
		}
	}
}

// runSubscriberService simulates a service that subscribes to events
func runSubscriberService(ctx context.Context, eventBus *eventbus.EventBusModule, stopChan <-chan os.Signal, tracker *EventTracker) {
	fmt.Println("📨 Subscriber Service started")

	// Subscribe to order events
	orderSub, err := eventBus.Subscribe(ctx, "order.*", func(ctx context.Context, event eventbus.Event) error {
		// Payload can be either a map (from NATS) or the original struct (from memory)
		var orderID interface{}
		if payloadMap, ok := event.Payload.(map[string]interface{}); ok {
			orderID = payloadMap["orderId"]
		} else if orderEvent, ok := event.Payload.(OrderEvent); ok {
			orderID = orderEvent.OrderID
		} else {
			fmt.Printf("📨 [ORDER SERVICE] Unknown payload type: %T\n", event.Payload)
			return nil
		}
		fmt.Printf("📨 [ORDER SERVICE] Processing order: %v\n", orderID)
		tracker.ConsumedOrder()
		return nil
	})
	if err != nil {
		fmt.Printf("Error subscribing to order events: %v\n", err)
		return
	}
	defer orderSub.Cancel()

	// Subscribe to analytics events asynchronously
	analyticsSub, err := eventBus.SubscribeAsync(ctx, "analytics.*", func(ctx context.Context, event eventbus.Event) error {
		// Payload can be either a map (from NATS) or the original struct (from memory)
		var eventType interface{}
		if payloadMap, ok := event.Payload.(map[string]interface{}); ok {
			eventType = payloadMap["eventType"]
		} else if analyticsEvent, ok := event.Payload.(AnalyticsEvent); ok {
			eventType = analyticsEvent.EventType
		} else {
			fmt.Printf("📨 [ANALYTICS SERVICE] Unknown payload type: %T\n", event.Payload)
			return nil
		}
		fmt.Printf("📨 [ANALYTICS SERVICE] Recording event: %v\n", eventType)
		tracker.ConsumedAnalytics()
		// Simulate some processing time
		time.Sleep(500 * time.Millisecond)
		return nil
	})
	if err != nil {
		fmt.Printf("Error subscribing to analytics events: %v\n", err)
		return
	}
	defer analyticsSub.Cancel()

	// Subscribe to notification events
	notifSub, err := eventBus.Subscribe(ctx, "notification.*", func(ctx context.Context, event eventbus.Event) error {
		// Payload can be either a map (from NATS) or the original struct (from memory)
		var message interface{}
		if payloadMap, ok := event.Payload.(map[string]interface{}); ok {
			message = payloadMap["message"]
		} else if notifEvent, ok := event.Payload.(NotificationEvent); ok {
			message = notifEvent.Message
		} else {
			fmt.Printf("📨 [NOTIFICATION SERVICE] Unknown payload type: %T\n", event.Payload)
			return nil
		}
		fmt.Printf("📨 [NOTIFICATION SERVICE] Sending notification: %v\n", message)
		tracker.ConsumedNotif()
		return nil
	})
	if err != nil {
		fmt.Printf("Error subscribing to notification events: %v\n", err)
		return
	}
	defer notifSub.Cancel()

	fmt.Println("✅ All subscriptions active")
	fmt.Println()

	// Wait for stop signal
	<-stopChan
	fmt.Println("📨 Subscriber Service stopping...")
}

func checkNATSAvailability() {
	fmt.Println("🔍 Checking NATS service availability:")

	// Check NATS connectivity
	natsAvailable := false
	if conn, err := net.DialTimeout("tcp", "localhost:4222", 2*time.Second); err == nil {
		conn.Close()
		natsAvailable = true
	}

	if natsAvailable {
		fmt.Println("  ✅ NATS service is reachable on localhost:4222")
		fmt.Println("  ✅ Ready for pub/sub messaging")
	} else {
		fmt.Println("  ❌ NATS service not reachable")
		fmt.Println("  💡 To enable NATS: docker-compose up -d")
	}
	fmt.Println()
}
