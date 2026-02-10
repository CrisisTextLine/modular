package eventbus

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/kinesis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/CrisisTextLine/modular/modules/eventbus/mocks"
)

// newTestKinesisEventBus creates a KinesisEventBus wired to a mock client,
// pre-started so Publish() can be called immediately.
func newTestKinesisEventBus(client KinesisClient) *KinesisEventBus {
	ctx, cancel := context.WithCancel(context.Background())
	return &KinesisEventBus{
		config: &KinesisConfig{
			StreamName: "test-stream",
			ShardCount: 1,
		},
		client:        client,
		subscriptions: make(map[string]map[string]*kinesisSubscription),
		ctx:           ctx,
		cancel:        cancel,
		isStarted:     true,
	}
}

func TestKinesisPublishPartitionKey(t *testing.T) {
	t.Run("uses context partition key when set", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		m := mocks.NewMockKinesisClient(ctrl)
		bus := newTestKinesisEventBus(m)
		defer bus.cancel()

		m.EXPECT().
			PutRecord(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, input *kinesis.PutRecordInput, optFns ...func(*kinesis.Options)) (*kinesis.PutRecordOutput, error) {
				assert.Equal(t, "user-42", *input.PartitionKey)
				assert.Equal(t, "test-stream", *input.StreamName)
				return &kinesis.PutRecordOutput{}, nil
			})

		ctx := WithPartitionKey(context.Background(), "user-42")
		err := bus.Publish(ctx, Event{Topic: "orders.created", Payload: "data"})
		require.NoError(t, err)
	})

	t.Run("falls back to topic when no context key", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		m := mocks.NewMockKinesisClient(ctrl)
		bus := newTestKinesisEventBus(m)
		defer bus.cancel()

		m.EXPECT().
			PutRecord(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, input *kinesis.PutRecordInput, optFns ...func(*kinesis.Options)) (*kinesis.PutRecordOutput, error) {
				assert.Equal(t, "orders.created", *input.PartitionKey)
				return &kinesis.PutRecordOutput{}, nil
			})

		err := bus.Publish(context.Background(), Event{Topic: "orders.created", Payload: "data"})
		require.NoError(t, err)
	})

	t.Run("falls back to topic when context key is empty string", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		m := mocks.NewMockKinesisClient(ctrl)
		bus := newTestKinesisEventBus(m)
		defer bus.cancel()

		m.EXPECT().
			PutRecord(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, input *kinesis.PutRecordInput, optFns ...func(*kinesis.Options)) (*kinesis.PutRecordOutput, error) {
				assert.Equal(t, "orders.created", *input.PartitionKey,
					"empty string partition key should fall back to topic for Kinesis")
				return &kinesis.PutRecordOutput{}, nil
			})

		ctx := WithPartitionKey(context.Background(), "")
		err := bus.Publish(ctx, Event{Topic: "orders.created", Payload: "data"})
		require.NoError(t, err)
	})

	t.Run("propagates PutRecord error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		m := mocks.NewMockKinesisClient(ctrl)
		bus := newTestKinesisEventBus(m)
		defer bus.cancel()

		m.EXPECT().
			PutRecord(gomock.Any(), gomock.Any()).
			Return(nil, fmt.Errorf("throttled"))

		err := bus.Publish(context.Background(), Event{Topic: "test", Payload: "data"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "throttled")
	})
}

func TestKinesisStart(t *testing.T) {
	t.Run("succeeds when stream already exists", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		m := mocks.NewMockKinesisClient(ctrl)
		bus := &KinesisEventBus{
			config:        &KinesisConfig{StreamName: "my-stream", ShardCount: 2},
			client:        m,
			subscriptions: make(map[string]map[string]*kinesisSubscription),
		}

		m.EXPECT().
			DescribeStream(gomock.Any(), gomock.Any()).
			Return(&kinesis.DescribeStreamOutput{}, nil)

		err := bus.Start(context.Background())
		require.NoError(t, err)
		assert.True(t, bus.isStarted)
	})

	t.Run("returns nil when already started", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		m := mocks.NewMockKinesisClient(ctrl)
		bus := newTestKinesisEventBus(m)
		defer bus.cancel()

		// No EXPECT calls — nothing should be called
		err := bus.Start(context.Background())
		require.NoError(t, err)
	})

	t.Run("returns error for invalid shard count when stream missing", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		m := mocks.NewMockKinesisClient(ctrl)
		bus := &KinesisEventBus{
			config:        &KinesisConfig{StreamName: "my-stream", ShardCount: 0},
			client:        m,
			subscriptions: make(map[string]map[string]*kinesisSubscription),
		}

		m.EXPECT().
			DescribeStream(gomock.Any(), gomock.Any()).
			Return(&kinesis.DescribeStreamOutput{}, fmt.Errorf("stream not found"))

		err := bus.Start(context.Background())
		assert.ErrorIs(t, err, ErrInvalidShardCount)
	})

	t.Run("propagates CreateStream error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		m := mocks.NewMockKinesisClient(ctrl)
		bus := &KinesisEventBus{
			config:        &KinesisConfig{StreamName: "my-stream", ShardCount: 2},
			client:        m,
			subscriptions: make(map[string]map[string]*kinesisSubscription),
		}

		m.EXPECT().
			DescribeStream(gomock.Any(), gomock.Any()).
			Return(&kinesis.DescribeStreamOutput{}, fmt.Errorf("stream not found"))
		m.EXPECT().
			CreateStream(gomock.Any(), gomock.Any()).
			Return(nil, fmt.Errorf("access denied"))

		err := bus.Start(context.Background())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "access denied")
		assert.False(t, bus.isStarted)
	})
}

func TestKinesisStop(t *testing.T) {
	t.Run("returns nil when not started", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		m := mocks.NewMockKinesisClient(ctrl)
		bus := &KinesisEventBus{
			config:        &KinesisConfig{StreamName: "test-stream"},
			client:        m,
			subscriptions: make(map[string]map[string]*kinesisSubscription),
		}

		err := bus.Stop(context.Background())
		require.NoError(t, err)
	})

	t.Run("clears subscriptions and marks stopped", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		m := mocks.NewMockKinesisClient(ctrl)
		bus := newTestKinesisEventBus(m)

		err := bus.Stop(context.Background())
		require.NoError(t, err)
		assert.False(t, bus.isStarted)
		assert.Empty(t, bus.subscriptions)
	})

	t.Run("returns timeout error when context expires", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		m := mocks.NewMockKinesisClient(ctrl)
		bus := newTestKinesisEventBus(m)

		// Add a wait group entry that never completes to simulate a stuck worker
		bus.wg.Add(1)

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		err := bus.Stop(ctx)
		assert.ErrorIs(t, err, ErrEventBusShutdownTimeout)

		// Clean up the stuck worker
		bus.wg.Done()
	})
}

func TestKinesisSubscribe(t *testing.T) {
	t.Run("returns error when not started", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		m := mocks.NewMockKinesisClient(ctrl)
		bus := &KinesisEventBus{
			config:        &KinesisConfig{StreamName: "test-stream"},
			client:        m,
			subscriptions: make(map[string]map[string]*kinesisSubscription),
		}

		_, err := bus.Subscribe(context.Background(), "topic", func(ctx context.Context, event Event) error { return nil })
		assert.ErrorIs(t, err, ErrEventBusNotStarted)
	})

	t.Run("returns error for nil handler", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		m := mocks.NewMockKinesisClient(ctrl)
		bus := newTestKinesisEventBus(m)
		defer bus.cancel()

		_, err := bus.Subscribe(context.Background(), "topic", nil)
		assert.ErrorIs(t, err, ErrEventHandlerNil)
	})
}

func TestKinesisPublishNotStarted(t *testing.T) {
	ctrl := gomock.NewController(t)
	m := mocks.NewMockKinesisClient(ctrl)
	bus := &KinesisEventBus{
		config:        &KinesisConfig{StreamName: "test-stream"},
		client:        m,
		subscriptions: make(map[string]map[string]*kinesisSubscription),
	}

	err := bus.Publish(context.Background(), Event{Topic: "test", Payload: "data"})
	assert.ErrorIs(t, err, ErrEventBusNotStarted)
}
