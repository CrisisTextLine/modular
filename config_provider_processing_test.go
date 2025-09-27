package modular

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestProcessMainConfig(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		hasProvider   bool
		enableVerbose bool
		expectConfig  bool
	}{
		{
			name:          "with provider and verbose enabled",
			hasProvider:   true,
			enableVerbose: true,
			expectConfig:  true,
		},
		{
			name:          "with provider and verbose disabled",
			hasProvider:   true,
			enableVerbose: false,
			expectConfig:  true,
		},
		{
			name:          "without provider",
			hasProvider:   false,
			enableVerbose: true,
			expectConfig:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockLogger := new(MockLogger)
			// Allow any debug calls - we don't care about specific messages
			mockLogger.On("Debug", mock.Anything, mock.Anything).Return().Maybe()

			app := &StdApplication{
				logger:      mockLogger,
				cfgSections: make(map[string]ConfigProvider),
			}

			if tt.hasProvider {
				app.cfgProvider = NewStdConfigProvider(&testCfg{Str: "test", Num: 42})
			}

			// Set up verbose config state
			app.verboseConfig = tt.enableVerbose

			cfgBuilder := NewConfig()
			tempConfigs := make(map[string]configInfo)

			result := processMainConfig(app, cfgBuilder, tempConfigs)

			assert.Equal(t, tt.expectConfig, result)
			if tt.expectConfig {
				assert.Contains(t, tempConfigs, "_main")
			} else {
				assert.NotContains(t, tempConfigs, "_main")
			}
		})
	}
}

func TestProcessSectionConfigs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		sections      map[string]ConfigProvider
		enableVerbose bool
		expectConfigs int
	}{
		{
			name: "with sections and verbose enabled",
			sections: map[string]ConfigProvider{
				"section1": NewStdConfigProvider(&testSectionCfg{Enabled: true, Name: "test"}),
				"section2": NewStdConfigProvider(&testSectionCfg{Enabled: false, Name: "test2"}),
			},
			enableVerbose: true,
			expectConfigs: 2,
		},
		{
			name: "with sections and verbose disabled",
			sections: map[string]ConfigProvider{
				"section1": NewStdConfigProvider(&testSectionCfg{Enabled: true, Name: "test"}),
			},
			enableVerbose: false,
			expectConfigs: 1,
		},
		{
			name:          "without sections",
			sections:      map[string]ConfigProvider{},
			enableVerbose: true,
			expectConfigs: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockLogger := new(MockLogger)
			// Allow any debug calls - we don't care about specific messages
			mockLogger.On("Debug", mock.Anything, mock.Anything).Return().Maybe()

			app := &StdApplication{
				logger:      mockLogger,
				cfgSections: tt.sections,
			}

			// Set up verbose config state
			app.verboseConfig = tt.enableVerbose

			cfgBuilder := NewConfig()
			tempConfigs := make(map[string]configInfo)

			result := processSectionConfigs(app, cfgBuilder, tempConfigs)

			assert.Equal(t, tt.expectConfigs > 0, result)
			assert.Len(t, tempConfigs, tt.expectConfigs)
		})
	}
}
