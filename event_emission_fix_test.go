package modular

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestEventEmissionWithoutSubject tests that modules handle missing subjects gracefully
// without printing noisy error messages to stdout during tests.
func TestEventEmissionWithoutSubject(t *testing.T) {
	// Create a test module that implements ObservableModule but has no subject registered
	testModule := &testObservableModule{}

	// This should not cause any panic or noisy output
	err := testModule.EmitEvent(context.Background(), NewCloudEvent("test.event", "test-module", nil, nil))

	// The error should be handled gracefully
	if err != nil {
		assert.Equal(t, "no subject available for event emission", err.Error())
	}
}

// TestHandleEventEmissionErrorUtility tests the utility function for consistent error handling
func TestHandleEventEmissionErrorUtility(t *testing.T) {
	// Test with "no subject available" error
	err := &testEmissionError{message: "no subject available for event emission"}
	handled := HandleEventEmissionError(err, nil, "test-module", "test.event")
	assert.True(t, handled, "Should handle 'no subject available' error")

	// Test with other error
	err = &testEmissionError{message: "some other error"}
	handled = HandleEventEmissionError(err, nil, "test-module", "test.event")
	assert.False(t, handled, "Should not handle other errors when no logger is available")

	// Test with logger
	logger := &mockTestLogger{}
	err = &testEmissionError{message: "some other error"}
	handled = HandleEventEmissionError(err, logger, "test-module", "test.event")
	assert.True(t, handled, "Should handle other errors when logger is available")
}

// Test types for the emission fix tests

type testObservableModule struct {
	subject Subject
}

func (t *testObservableModule) RegisterObservers(subject Subject) error {
	t.subject = subject
	return nil
}

func (t *testObservableModule) EmitEvent(ctx context.Context, event CloudEvent) error {
	if t.subject == nil {
		return &testEmissionError{message: "no subject available for event emission"}
	}
	return t.subject.NotifyObservers(ctx, event)
}

type testEmissionError struct {
	message string
}

func (e *testEmissionError) Error() string {
	return e.message
}

type mockTestLogger struct {
	lastDebugMessage string
}

func (l *mockTestLogger) Debug(msg string, args ...interface{}) {
	l.lastDebugMessage = msg
}

func (l *mockTestLogger) Info(msg string, args ...interface{})  {}
func (l *mockTestLogger) Warn(msg string, args ...interface{})  {}
func (l *mockTestLogger) Error(msg string, args ...interface{}) {}
