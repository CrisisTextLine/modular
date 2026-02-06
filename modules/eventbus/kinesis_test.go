package eventbus

import (
	"errors"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/kinesis/types"
	"github.com/stretchr/testify/assert"
)

func TestIsExpiredIteratorError(t *testing.T) {
	t.Run("returns true for ExpiredIteratorException", func(t *testing.T) {
		msg := "Iterator expired"
		err := &types.ExpiredIteratorException{Message: &msg}
		assert.True(t, isExpiredIteratorError(err))
	})

	t.Run("returns true for wrapped ExpiredIteratorException", func(t *testing.T) {
		msg := "Iterator expired"
		inner := &types.ExpiredIteratorException{Message: &msg}
		wrapped := fmt.Errorf("kinesis error: %w", inner)
		assert.True(t, isExpiredIteratorError(wrapped))
	})

	t.Run("returns false for other errors", func(t *testing.T) {
		assert.False(t, isExpiredIteratorError(errors.New("something else")))
	})

	t.Run("returns false for other Kinesis errors", func(t *testing.T) {
		msg := "Throughput exceeded"
		err := &types.ProvisionedThroughputExceededException{Message: &msg}
		assert.False(t, isExpiredIteratorError(err))
	})
}
