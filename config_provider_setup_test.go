package modular

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Test implementation of ConfigSetup
type testSetupCfg struct {
	Value       string `yaml:"value"`
	setupCalled bool
	shouldError bool
}

func (t *testSetupCfg) Setup() error {
	t.setupCalled = true
	if t.shouldError {
		return ErrSetupFailed
	}
	return nil
}

func TestConfig_Feed(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		setupConfig    func() (*Config, *MockComplexFeeder)
		expectFeedErr  bool
		expectKeyErr   bool
		expectedErrMsg string
	}{
		{
			name: "successful feed",
			setupConfig: func() (*Config, *MockComplexFeeder) {
				cfg := NewConfig()
				feeder := new(MockComplexFeeder)
				feeder.On("Feed", mock.Anything).Return(nil)
				feeder.On("FeedKey", "main", mock.Anything).Return(nil)
				feeder.On("FeedKey", "test", mock.Anything).Return(nil)
				cfg.AddFeeder(feeder)
				cfg.AddStructKey("main", &testCfg{})
				cfg.AddStructKey("test", &testCfg{})
				return cfg, feeder
			},
			expectFeedErr: false,
		},
		{
			name: "feed error",
			setupConfig: func() (*Config, *MockComplexFeeder) {
				cfg := NewConfig()
				feeder := new(MockComplexFeeder)
				feeder.On("Feed", mock.Anything).Return(ErrFeedFailed)
				cfg.AddFeeder(feeder)
				cfg.AddStructKey("main", &testCfg{})
				return cfg, feeder
			},
			expectFeedErr:  true,
			expectedErrMsg: "feed error",
		},
		{
			name: "feedKey error",
			setupConfig: func() (*Config, *MockComplexFeeder) {
				cfg := NewConfig()
				feeder := new(MockComplexFeeder)
				feeder.On("Feed", mock.Anything).Return(nil)
				// Due to map iteration order being random, either key could be processed first
				// If "test" is processed first, it will fail and stop processing
				// If "main" is processed first, it will succeed, then "test" will fail
				feeder.On("FeedKey", "main", mock.Anything).Return(nil).Maybe()
				feeder.On("FeedKey", "test", mock.Anything).Return(ErrFeedKeyFailed)
				cfg.AddFeeder(feeder)
				cfg.AddStructKey("main", &testCfg{})
				cfg.AddStructKey("test", &testCfg{})
				return cfg, feeder
			},
			expectFeedErr:  true,
			expectKeyErr:   true,
			expectedErrMsg: "feeder error",
		},
		{
			name: "setup success",
			setupConfig: func() (*Config, *MockComplexFeeder) {
				cfg := NewConfig()
				feeder := new(MockComplexFeeder)
				feeder.On("Feed", mock.Anything).Return(nil)
				feeder.On("FeedKey", "main", mock.Anything).Return(nil)
				feeder.On("FeedKey", "test", mock.Anything).Return(nil)
				cfg.AddFeeder(feeder)
				cfg.AddStructKey("main", &testCfg{})
				cfg.AddStructKey("test", &testSetupCfg{})
				return cfg, feeder
			},
			expectFeedErr: false,
		},
		{
			name: "setup error",
			setupConfig: func() (*Config, *MockComplexFeeder) {
				cfg := NewConfig()
				feeder := new(MockComplexFeeder)
				feeder.On("Feed", mock.Anything).Return(nil)
				// Due to map iteration order being random, either key could be processed first
				// If "test" is processed first, it will succeed then fail at setup
				// If "main" is processed first, it will succeed, then "test" will succeed and fail at setup
				feeder.On("FeedKey", "main", mock.Anything).Return(nil).Maybe()
				feeder.On("FeedKey", "test", mock.Anything).Return(nil).Maybe()
				cfg.AddFeeder(feeder)
				cfg.AddStructKey("main", &testCfg{})
				cfg.AddStructKey("test", &testSetupCfg{shouldError: true})
				return cfg, feeder
			},
			expectFeedErr:  true,
			expectedErrMsg: "config setup error for test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, feeder := tt.setupConfig()

			err := cfg.Feed()

			if tt.expectFeedErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrMsg)
			} else {
				require.NoError(t, err)
				// Check if setup was called when using testSetupCfg
				if setupCfg, ok := cfg.StructKeys["test"].(*testSetupCfg); ok {
					assert.True(t, setupCfg.setupCalled)
				}
			}

			feeder.AssertExpectations(t)
		})
	}
}
