package eventbus

import "context"

// partitionKeyCtxKey is the context key for partition key routing hints.
type partitionKeyCtxKey struct{}

// WithPartitionKey returns a context with a partition key routing hint.
//
// The partition key controls how events are distributed across shards/partitions:
//   - Kinesis: determines which shard receives the record (default: topic name)
//   - Kafka: determines which partition receives the message (using the client's default partitioner)
//   - Memory, Redis, NATS: ignored (no partitioning concept)
//
// Note: an empty string key is treated as unset for Kinesis (falls back to topic)
// but is honored as-is for Kafka.
//
// Use this when you want related events to be routed to the same shard/partition
// so that broker-level ordering within that shard/partition is preserved. The
// actual handler processing order still depends on the subscription/consumer
// behavior (for example, concurrency and async processing settings).
//
// Example:
//
//	// Ensure all events for a user go to the same shard
//	ctx = eventbus.WithPartitionKey(ctx, userID)
//	err := eventBus.Publish(ctx, "user.action", actionData)
func WithPartitionKey(ctx context.Context, key string) context.Context {
	return context.WithValue(ctx, partitionKeyCtxKey{}, key)
}

// PartitionKeyFromContext extracts the partition key from a context.
// Returns the key and true if set, or empty string and false if not set.
func PartitionKeyFromContext(ctx context.Context) (string, bool) {
	key, ok := ctx.Value(partitionKeyCtxKey{}).(string)
	return key, ok
}
