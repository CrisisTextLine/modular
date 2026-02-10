package eventbus

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/IBM/sarama"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/CrisisTextLine/modular/modules/eventbus/mocks"
)

// newTestKafkaEventBus creates a KafkaEventBus wired to a mock producer,
// pre-started so Publish() can be called immediately.
func newTestKafkaEventBus(producer sarama.SyncProducer) *KafkaEventBus {
	ctx, cancel := context.WithCancel(context.Background())
	return &KafkaEventBus{
		config: &KafkaConfig{
			Brokers: []string{"localhost:9092"},
			GroupID: "test-group",
		},
		producer:      producer,
		subscriptions: make(map[string]map[string]*kafkaSubscription),
		ctx:           ctx,
		cancel:        cancel,
		isStarted:     true,
	}
}

func TestKafkaPublishPartitionKey(t *testing.T) {
	t.Run("sets message key from context partition key", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		m := mocks.NewMockSyncProducer(ctrl)
		bus := newTestKafkaEventBus(m)
		defer bus.cancel()

		m.EXPECT().
			SendMessage(gomock.Any()).
			DoAndReturn(func(msg *sarama.ProducerMessage) (int32, int64, error) {
				assert.Equal(t, "orders.created", msg.Topic)
				keyBytes, err := msg.Key.Encode()
				require.NoError(t, err)
				assert.Equal(t, "user-42", string(keyBytes))
				return 0, 0, nil
			})

		ctx := WithPartitionKey(context.Background(), "user-42")
		err := bus.Publish(ctx, Event{Topic: "orders.created", Payload: "data"})
		require.NoError(t, err)
	})

	t.Run("message key is nil when no partition key in context", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		m := mocks.NewMockSyncProducer(ctrl)
		bus := newTestKafkaEventBus(m)
		defer bus.cancel()

		m.EXPECT().
			SendMessage(gomock.Any()).
			DoAndReturn(func(msg *sarama.ProducerMessage) (int32, int64, error) {
				assert.Nil(t, msg.Key, "message key should be nil when no partition key set")
				return 0, 0, nil
			})

		err := bus.Publish(context.Background(), Event{Topic: "orders.created", Payload: "data"})
		require.NoError(t, err)
	})

	t.Run("empty string partition key is honored", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		m := mocks.NewMockSyncProducer(ctrl)
		bus := newTestKafkaEventBus(m)
		defer bus.cancel()

		m.EXPECT().
			SendMessage(gomock.Any()).
			DoAndReturn(func(msg *sarama.ProducerMessage) (int32, int64, error) {
				require.NotNil(t, msg.Key, "empty string should be honored as a key for Kafka")
				keyBytes, err := msg.Key.Encode()
				require.NoError(t, err)
				assert.Equal(t, "", string(keyBytes))
				return 0, 0, nil
			})

		ctx := WithPartitionKey(context.Background(), "")
		err := bus.Publish(ctx, Event{Topic: "orders.created", Payload: "data"})
		require.NoError(t, err)
	})

	t.Run("propagates SendMessage error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		m := mocks.NewMockSyncProducer(ctrl)
		bus := newTestKafkaEventBus(m)
		defer bus.cancel()

		m.EXPECT().
			SendMessage(gomock.Any()).
			Return(int32(0), int64(0), fmt.Errorf("broker unavailable"))

		err := bus.Publish(context.Background(), Event{Topic: "test", Payload: "data"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "broker unavailable")
	})
}

func TestKafkaStart(t *testing.T) {
	t.Run("sets started state", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		producer := mocks.NewMockSyncProducer(ctrl)
		bus := &KafkaEventBus{
			config:        &KafkaConfig{Brokers: []string{"localhost:9092"}, GroupID: "test"},
			producer:      producer,
			subscriptions: make(map[string]map[string]*kafkaSubscription),
		}

		err := bus.Start(context.Background())
		require.NoError(t, err)
		assert.True(t, bus.isStarted)
	})

	t.Run("returns nil when already started", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		producer := mocks.NewMockSyncProducer(ctrl)
		bus := newTestKafkaEventBus(producer)
		defer bus.cancel()

		err := bus.Start(context.Background())
		require.NoError(t, err)
	})
}

func TestKafkaStop(t *testing.T) {
	t.Run("returns nil when not started", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		producer := mocks.NewMockSyncProducer(ctrl)
		bus := &KafkaEventBus{
			config:        &KafkaConfig{Brokers: []string{"localhost:9092"}, GroupID: "test"},
			producer:      producer,
			subscriptions: make(map[string]map[string]*kafkaSubscription),
		}

		err := bus.Stop(context.Background())
		require.NoError(t, err)
	})

	t.Run("closes producer and consumer group", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		producer := mocks.NewMockSyncProducer(ctrl)
		consumerGroup := mocks.NewMockConsumerGroup(ctrl)

		ctx, cancel := context.WithCancel(context.Background())
		bus := &KafkaEventBus{
			config:        &KafkaConfig{Brokers: []string{"localhost:9092"}, GroupID: "test"},
			producer:      producer,
			consumerGroup: consumerGroup,
			subscriptions: make(map[string]map[string]*kafkaSubscription),
			ctx:           ctx,
			cancel:        cancel,
			isStarted:     true,
		}

		producer.EXPECT().Close().Return(nil)
		consumerGroup.EXPECT().Close().Return(nil)

		err := bus.Stop(context.Background())
		require.NoError(t, err)
		assert.False(t, bus.isStarted)
	})

	t.Run("propagates producer close error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		producer := mocks.NewMockSyncProducer(ctrl)
		consumerGroup := mocks.NewMockConsumerGroup(ctrl)

		ctx, cancel := context.WithCancel(context.Background())
		bus := &KafkaEventBus{
			config:        &KafkaConfig{Brokers: []string{"localhost:9092"}, GroupID: "test"},
			producer:      producer,
			consumerGroup: consumerGroup,
			subscriptions: make(map[string]map[string]*kafkaSubscription),
			ctx:           ctx,
			cancel:        cancel,
			isStarted:     true,
		}

		producer.EXPECT().Close().Return(fmt.Errorf("producer close failed"))

		err := bus.Stop(context.Background())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "producer close failed")
	})

	t.Run("propagates consumer group close error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		producer := mocks.NewMockSyncProducer(ctrl)
		consumerGroup := mocks.NewMockConsumerGroup(ctrl)

		ctx, cancel := context.WithCancel(context.Background())
		bus := &KafkaEventBus{
			config:        &KafkaConfig{Brokers: []string{"localhost:9092"}, GroupID: "test"},
			producer:      producer,
			consumerGroup: consumerGroup,
			subscriptions: make(map[string]map[string]*kafkaSubscription),
			ctx:           ctx,
			cancel:        cancel,
			isStarted:     true,
		}

		producer.EXPECT().Close().Return(nil)
		consumerGroup.EXPECT().Close().Return(fmt.Errorf("consumer group close failed"))

		err := bus.Stop(context.Background())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "consumer group close failed")
	})

	t.Run("returns timeout error when context expires", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		producer := mocks.NewMockSyncProducer(ctrl)

		ctx, cancel := context.WithCancel(context.Background())
		bus := &KafkaEventBus{
			config:        &KafkaConfig{Brokers: []string{"localhost:9092"}, GroupID: "test"},
			producer:      producer,
			subscriptions: make(map[string]map[string]*kafkaSubscription),
			ctx:           ctx,
			cancel:        cancel,
			isStarted:     true,
		}

		// Add a wait group entry that never completes
		bus.wg.Add(1)

		stopCtx, stopCancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer stopCancel()

		err := bus.Stop(stopCtx)
		assert.ErrorIs(t, err, ErrEventBusShutdownTimeout)

		bus.wg.Done()
	})
}

func TestKafkaSubscribe(t *testing.T) {
	t.Run("returns error when not started", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		producer := mocks.NewMockSyncProducer(ctrl)
		bus := &KafkaEventBus{
			config:        &KafkaConfig{Brokers: []string{"localhost:9092"}, GroupID: "test"},
			producer:      producer,
			subscriptions: make(map[string]map[string]*kafkaSubscription),
		}

		_, err := bus.Subscribe(context.Background(), "topic", func(ctx context.Context, event Event) error { return nil })
		assert.ErrorIs(t, err, ErrEventBusNotStarted)
	})

	t.Run("returns error for nil handler", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		producer := mocks.NewMockSyncProducer(ctrl)
		bus := newTestKafkaEventBus(producer)
		defer bus.cancel()

		_, err := bus.Subscribe(context.Background(), "topic", nil)
		assert.ErrorIs(t, err, ErrEventHandlerNil)
	})
}

func TestKafkaPublishNotStarted(t *testing.T) {
	ctrl := gomock.NewController(t)
	producer := mocks.NewMockSyncProducer(ctrl)
	bus := &KafkaEventBus{
		config:        &KafkaConfig{Brokers: []string{"localhost:9092"}, GroupID: "test"},
		producer:      producer,
		subscriptions: make(map[string]map[string]*kafkaSubscription),
	}

	err := bus.Publish(context.Background(), Event{Topic: "test", Payload: "data"})
	assert.ErrorIs(t, err, ErrEventBusNotStarted)
}
