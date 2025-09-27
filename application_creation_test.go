package modular

import (
	"testing"
)

func TestNewApplication(t *testing.T) {
	type args struct {
		cfgProvider ConfigProvider
		logger      Logger
	}
	cp := NewStdConfigProvider(testCfg{Str: "test"})
	log := &logger{}
	tests := []struct {
		name           string
		args           args
		expectedLogger Logger
	}{
		{
			name: "TestNewApplication",
			args: args{
				cfgProvider: nil,
				logger:      nil,
			},
			expectedLogger: nil,
		},
		{
			name: "TestNewApplicationWithConfigProviderAndLogger",
			args: args{
				cfgProvider: cp,
				logger:      log,
			},
			expectedLogger: log,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewStdApplication(tt.args.cfgProvider, tt.args.logger)

			// Test functional properties
			if got.ConfigProvider() != tt.args.cfgProvider {
				t.Errorf("NewStdApplication().ConfigProvider() = %v, want %v", got.ConfigProvider(), tt.args.cfgProvider)
			}

			if got.Logger() != tt.expectedLogger {
				t.Errorf("NewStdApplication().Logger() = %v, want %v", got.Logger(), tt.expectedLogger)
			}

			// Check that logger service is properly registered
			svcRegistry := got.SvcRegistry()
			if svcRegistry["logger"] != tt.expectedLogger {
				t.Errorf("NewStdApplication() logger service = %v, want %v", svcRegistry["logger"], tt.expectedLogger)
			}

			// Verify config sections is initialized (empty map)
			if len(got.ConfigSections()) != 0 {
				t.Errorf("NewStdApplication().ConfigSections() should be empty, got %v", got.ConfigSections())
			}
		})
	}
}
