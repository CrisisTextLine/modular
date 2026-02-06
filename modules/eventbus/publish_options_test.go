package eventbus

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/CrisisTextLine/modular"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithPartitionKey(t *testing.T) {
	t.Run("sets partition key in context", func(t *testing.T) {
		ctx := context.Background()
		ctx = WithPartitionKey(ctx, "user-123")

		key, ok := PartitionKeyFromContext(ctx)
		require.True(t, ok)
		assert.Equal(t, "user-123", key)
	})

	t.Run("returns false when not set", func(t *testing.T) {
		ctx := context.Background()

		key, ok := PartitionKeyFromContext(ctx)
		assert.False(t, ok)
		assert.Equal(t, "", key)
	})

	t.Run("empty string is valid", func(t *testing.T) {
		ctx := context.Background()
		ctx = WithPartitionKey(ctx, "")

		key, ok := PartitionKeyFromContext(ctx)
		require.True(t, ok)
		assert.Equal(t, "", key)
	})

	t.Run("later call overrides earlier", func(t *testing.T) {
		ctx := context.Background()
		ctx = WithPartitionKey(ctx, "first")
		ctx = WithPartitionKey(ctx, "second")

		key, ok := PartitionKeyFromContext(ctx)
		require.True(t, ok)
		assert.Equal(t, "second", key)
	})
}

func TestPublishWithPartitionKey(t *testing.T) {
	t.Run("publish with partition key succeeds", func(t *testing.T) {
		module := NewModule().(*EventBusModule)
		app := newMockApp()

		cfg := &EventBusConfig{
			Engine:                 "memory",
			MaxEventQueueSize:      100,
			DefaultEventBufferSize: 10,
			WorkerCount:            2,
		}
		app.RegisterConfigSection(ModuleName, modular.NewStdConfigProvider(cfg))

		err := module.Init(app)
		require.NoError(t, err)

		ctx := context.Background()
		err = module.Start(ctx)
		require.NoError(t, err)
		defer func() {
			_ = module.Stop(ctx)
		}()

		eventReceived := make(chan Event, 1)
		_, err = module.Subscribe(ctx, "test.partitioned", func(ctx context.Context, event Event) error {
			eventReceived <- event
			return nil
		})
		require.NoError(t, err)

		// Use context to set partition key
		pubCtx := WithPartitionKey(ctx, "custom-key")
		err = module.Publish(pubCtx, "test.partitioned", "test-payload")
		require.NoError(t, err)

		select {
		case event := <-eventReceived:
			assert.Equal(t, "test-payload", event.Payload)
			assert.Equal(t, "test.partitioned", event.Topic)
		case <-time.After(2 * time.Second):
			t.Fatal("Event not received within timeout")
		}
	})

	t.Run("publish without partition key still works", func(t *testing.T) {
		module := NewModule().(*EventBusModule)
		app := newMockApp()

		cfg := &EventBusConfig{
			Engine:                 "memory",
			MaxEventQueueSize:      100,
			DefaultEventBufferSize: 10,
			WorkerCount:            2,
		}
		app.RegisterConfigSection(ModuleName, modular.NewStdConfigProvider(cfg))

		err := module.Init(app)
		require.NoError(t, err)

		ctx := context.Background()
		err = module.Start(ctx)
		require.NoError(t, err)
		defer func() {
			_ = module.Stop(ctx)
		}()

		eventReceived := make(chan Event, 1)
		_, err = module.Subscribe(ctx, "test.basic", func(ctx context.Context, event Event) error {
			eventReceived <- event
			return nil
		})
		require.NoError(t, err)

		// Publish without partition key (backward compatible)
		err = module.Publish(ctx, "test.basic", "basic-payload")
		require.NoError(t, err)

		select {
		case event := <-eventReceived:
			assert.Equal(t, "basic-payload", event.Payload)
		case <-time.After(2 * time.Second):
			t.Fatal("Event not received within timeout")
		}
	})
}

func TestPublishWithPartitionKeyConcurrency(t *testing.T) {
	t.Run("concurrent publishes with different partition keys", func(t *testing.T) {
		module := NewModule().(*EventBusModule)
		app := newMockApp()

		cfg := &EventBusConfig{
			Engine:                 "memory",
			MaxEventQueueSize:      1000,
			DefaultEventBufferSize: 100,
			WorkerCount:            5,
		}
		app.RegisterConfigSection(ModuleName, modular.NewStdConfigProvider(cfg))

		err := module.Init(app)
		require.NoError(t, err)

		ctx := context.Background()
		err = module.Start(ctx)
		require.NoError(t, err)
		defer func() {
			_ = module.Stop(ctx)
		}()

		var receivedCount int64
		var mu sync.Mutex
		_, err = module.SubscribeAsync(ctx, "concurrent.topic", func(ctx context.Context, event Event) error {
			mu.Lock()
			receivedCount++
			mu.Unlock()
			return nil
		})
		require.NoError(t, err)

		const numPublishers = 50
		const messagesPerPublisher = 10
		var wg sync.WaitGroup

		for i := 0; i < numPublishers; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				for j := 0; j < messagesPerPublisher; j++ {
					key := string(rune('a' + (idx % 10)))
					pubCtx := WithPartitionKey(ctx, key)
					pubErr := module.Publish(pubCtx, "concurrent.topic", idx*100+j)
					assert.NoError(t, pubErr)
				}
			}(i)
		}

		wg.Wait()
	})
}
