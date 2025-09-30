package modular

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	updatedValue = "updated"
)

type testCfg struct {
	Str string `yaml:"str"`
	Num int    `yaml:"num"`
}

type testSectionCfg struct {
	Enabled bool   `yaml:"enabled"`
	Name    string `yaml:"name"`
}

// Mock for ComplexFeeder
type MockComplexFeeder struct {
	mock.Mock
}

func (m *MockComplexFeeder) Feed(structure interface{}) error {
	args := m.Called(structure)
	if err := args.Error(0); err != nil {
		return fmt.Errorf("mock feeder error: %w", err)
	}
	return nil
}

func (m *MockComplexFeeder) FeedKey(key string, target interface{}) error {
	args := m.Called(key, target)
	if err := args.Error(0); err != nil {
		return fmt.Errorf("mock feeder key error: %w", err)
	}
	return nil
}

func TestNewStdConfigProvider(t *testing.T) {
	t.Parallel()
	cfg := &testCfg{Str: "test", Num: 42}
	provider := NewStdConfigProvider(cfg)

	assert.NotNil(t, provider)
	assert.Equal(t, cfg, provider.GetConfig())
}

func TestStdConfigProvider_GetConfig(t *testing.T) {
	t.Parallel()
	cfg := &testCfg{Str: "test", Num: 42}
	provider := &StdConfigProvider{cfg: cfg}

	assert.Equal(t, cfg, provider.GetConfig())
}

func TestNewConfig(t *testing.T) {
	t.Parallel()
	cfg := NewConfig()

	assert.NotNil(t, cfg)
	assert.NotNil(t, cfg.Feeders)
	assert.NotNil(t, cfg.StructKeys)
	assert.Empty(t, cfg.StructKeys)
}

func TestConfig_AddStructKey(t *testing.T) {
	t.Parallel()
	cfg := NewConfig()
	target := &testCfg{}

	result := cfg.AddStructKey("test", target)

	assert.Equal(t, cfg, result)
	assert.Len(t, cfg.StructKeys, 1)
	assert.Equal(t, target, cfg.StructKeys["test"])
}

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

func Test_createTempConfig(t *testing.T) {
	t.Parallel()
	t.Run("with pointer", func(t *testing.T) {
		originalCfg := &testCfg{Str: "test", Num: 42}
		tempCfg, info, err := createTempConfig(originalCfg)

		require.NoError(t, err)
		require.NotNil(t, tempCfg)
		assert.True(t, info.isPtr)
		assert.Equal(t, reflect.ValueOf(originalCfg).Type(), info.tempVal.Type())
	})

	t.Run("with non-pointer", func(t *testing.T) {
		originalCfg := testCfg{Str: "test", Num: 42}
		tempCfg, info, err := createTempConfig(originalCfg)

		require.NoError(t, err)
		require.NotNil(t, tempCfg)
		assert.False(t, info.isPtr)
		assert.Equal(t, reflect.PointerTo(reflect.ValueOf(originalCfg).Type()), info.tempVal.Type())
	})

	t.Run("maps and slices are deeply copied", func(t *testing.T) {
		type ConfigWithMaps struct {
			Name     string
			Settings map[string]string
			Tags     []string
		}

		// Create an original config with initialized maps and slices
		originalCfg := &ConfigWithMaps{
			Name: "original",
			Settings: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
			Tags: []string{"tag1", "tag2"},
		}

		// Create temp config
		tempCfg, info, err := createTempConfig(originalCfg)
		require.NoError(t, err)
		require.NotNil(t, tempCfg)

		tempCfgTyped := tempCfg.(*ConfigWithMaps)

		// Verify initial values are copied
		assert.Equal(t, "original", tempCfgTyped.Name)
		assert.Equal(t, "value1", tempCfgTyped.Settings["key1"])
		assert.Equal(t, "tag1", tempCfgTyped.Tags[0])

		// Modify the temp config's maps and slices
		tempCfgTyped.Settings["key1"] = "MODIFIED"
		tempCfgTyped.Settings["newkey"] = "newvalue"
		tempCfgTyped.Tags[0] = "MODIFIED"

		// Original should NOT be affected (deep copy ensures isolation)
		assert.Equal(t, "value1", originalCfg.Settings["key1"],
			"Original config's map should not be affected by temp config modifications")
		assert.NotContains(t, originalCfg.Settings, "newkey",
			"Original config's map should not be affected by temp config modifications")
		assert.Equal(t, "tag1", originalCfg.Tags[0],
			"Original config's slice should not be affected by temp config modifications")

		// Verify the info struct is correct
		assert.True(t, info.isPtr)
	})
}

func Test_updateConfig(t *testing.T) {
	t.Parallel()
	t.Run("with pointer config", func(t *testing.T) {
		originalCfg := &testCfg{Str: "old", Num: 0}
		tempCfg := &testCfg{Str: "new", Num: 42}

		mockLogger := new(MockLogger)
		app := &StdApplication{logger: mockLogger}

		origInfo := configInfo{
			originalVal: reflect.ValueOf(originalCfg),
			tempVal:     reflect.ValueOf(tempCfg),
			isPtr:       true,
		}

		updateConfig(app, origInfo)

		// Check the original config was updated
		assert.Equal(t, "new", originalCfg.Str)
		assert.Equal(t, 42, originalCfg.Num)
	})

	t.Run("with non-pointer config", func(t *testing.T) {
		originalCfg := testCfg{Str: "old", Num: 0}
		tempCfgPtr, origInfo, err := createTempConfig(originalCfg)
		require.NoError(t, err)
		tempCfgPtr.(*testCfg).Str = "new"
		tempCfgPtr.(*testCfg).Num = 42

		mockLogger := new(MockLogger)
		mockLogger.On("Debug",
			"Creating new provider with updated config (original was non-pointer)",
			[]interface{}(nil)).Return()
		app := &StdApplication{
			logger:      mockLogger,
			cfgProvider: NewStdConfigProvider(originalCfg),
		}

		updateConfig(app, origInfo)

		// Check the updated provider from the app (not the original provider reference)
		updated := app.cfgProvider.GetConfig()
		assert.Equal(t, reflect.Struct, reflect.ValueOf(updated).Kind())
		assert.Equal(t, "new", updated.(testCfg).Str)
		assert.Equal(t, 42, updated.(testCfg).Num)
		mockLogger.AssertExpectations(t)
	})
}

func Test_updateSectionConfig(t *testing.T) {
	t.Parallel()
	t.Run("with pointer section config", func(t *testing.T) {
		originalCfg := &testSectionCfg{Enabled: false, Name: "old"}
		tempCfg := &testSectionCfg{Enabled: true, Name: "new"}

		mockLogger := new(MockLogger)
		app := &StdApplication{
			logger:      mockLogger,
			cfgSections: make(map[string]ConfigProvider),
		}
		app.cfgSections["test"] = NewStdConfigProvider(originalCfg)

		sectionInfo := configInfo{
			originalVal: reflect.ValueOf(originalCfg),
			tempVal:     reflect.ValueOf(tempCfg),
			isPtr:       true,
		}

		updateSectionConfig(app, "test", sectionInfo)

		// Check the original config was updated
		assert.True(t, originalCfg.Enabled)
		assert.Equal(t, "new", originalCfg.Name)
	})

	t.Run("with non-pointer section config", func(t *testing.T) {
		originalCfg := testSectionCfg{Enabled: false, Name: "old"}
		tempCfgPtr, sectionInfo, err := createTempConfig(originalCfg)
		require.NoError(t, err)

		// Cast and update the temp config
		tempCfgPtr.(*testSectionCfg).Enabled = true
		tempCfgPtr.(*testSectionCfg).Name = "new"

		mockLogger := new(MockLogger)
		mockLogger.On("Debug", "Creating new provider for section", []interface{}{"section", "test"}).Return()

		app := &StdApplication{
			logger:      mockLogger,
			cfgSections: make(map[string]ConfigProvider),
		}
		app.cfgSections["test"] = NewStdConfigProvider(originalCfg)

		updateSectionConfig(app, "test", sectionInfo)

		// Check a new provider was created
		sectCfg := app.cfgSections["test"].GetConfig()
		assert.True(t, sectCfg.(testSectionCfg).Enabled)
		assert.Equal(t, "new", sectCfg.(testSectionCfg).Name)
		mockLogger.AssertExpectations(t)
	})
}

func Test_loadAppConfig(t *testing.T) {
	t.Parallel()
	// Tests now rely on per-application feeders (SetConfigFeeders) instead of mutating
	// the global ConfigFeeders slice to support safe parallelization.

	tests := []struct {
		name           string
		setupApp       func() *StdApplication
		setupFeeders   func() []Feeder
		expectError    bool
		validateResult func(t *testing.T, app *StdApplication)
	}{
		{
			name: "successful config load",
			setupApp: func() *StdApplication {
				mockLogger := new(MockLogger)
				mockLogger.On("Debug", "Added main config for loading", mock.Anything).Return()
				mockLogger.On("Debug", "Added section config for loading", mock.Anything).Return()
				mockLogger.On("Debug", "Updated main config", mock.Anything).Return()
				mockLogger.On("Debug", "Updated section config", mock.Anything).Return()

				app := &StdApplication{
					logger:      mockLogger,
					cfgProvider: NewStdConfigProvider(&testCfg{Str: "old", Num: 0}),
					cfgSections: make(map[string]ConfigProvider),
				}
				app.cfgSections["section1"] = NewStdConfigProvider(&testSectionCfg{Enabled: false, Name: "old"})
				return app
			},
			setupFeeders: func() []Feeder {
				feeder := new(MockComplexFeeder)
				// Setup to handle any Feed call - let the Run function determine the type
				feeder.On("Feed", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
					if cfg, ok := args.Get(0).(*testCfg); ok {
						cfg.Str = updatedValue
						cfg.Num = 42
					} else if cfg, ok := args.Get(0).(*testSectionCfg); ok {
						cfg.Enabled = true
						cfg.Name = "updated"
					}
				})
				// Setup for main config FeedKey calls
				feeder.On("FeedKey", "_main", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
					cfg := args.Get(1).(*testCfg)
					cfg.Str = updatedValue
					cfg.Num = 42
				})
				// Setup for section config FeedKey calls
				feeder.On("FeedKey", "section1", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
					cfg := args.Get(1).(*testSectionCfg)
					cfg.Enabled = true
					cfg.Name = "updated"
				})
				return []Feeder{feeder}
			},
			expectError: false,
			validateResult: func(t *testing.T, app *StdApplication) {
				mainCfg := app.cfgProvider.GetConfig().(*testCfg)
				assert.Equal(t, updatedValue, mainCfg.Str)
				assert.Equal(t, 42, mainCfg.Num)

				sectionCfg := app.cfgSections["section1"].GetConfig().(*testSectionCfg)
				assert.True(t, sectionCfg.Enabled)
				assert.Equal(t, "updated", sectionCfg.Name)
			},
		},
		{
			name: "feed error",
			setupApp: func() *StdApplication {
				mockLogger := new(MockLogger)
				mockLogger.On("Debug", "Added main config for loading", mock.Anything).Return()
				app := &StdApplication{
					logger:      mockLogger,
					cfgProvider: NewStdConfigProvider(&testCfg{Str: "old", Num: 0}),
					cfgSections: make(map[string]ConfigProvider),
				}
				return app
			},
			setupFeeders: func() []Feeder {
				feeder := new(MockComplexFeeder)
				feeder.On("Feed", mock.Anything).Return(ErrFeedFailed)
				return []Feeder{feeder}
			},
			expectError: true,
			validateResult: func(t *testing.T, app *StdApplication) {
				// Config should remain unchanged
				mainCfg := app.cfgProvider.GetConfig().(*testCfg)
				assert.Equal(t, "old", mainCfg.Str)
				assert.Equal(t, 0, mainCfg.Num)
			},
		},
		{
			name: "feedKey error",
			setupApp: func() *StdApplication {
				mockLogger := new(MockLogger)
				mockLogger.On("Debug", "Added main config for loading", mock.Anything).Return()
				mockLogger.On("Debug", "Added section config for loading", mock.Anything).Return()
				app := &StdApplication{
					logger:      mockLogger,
					cfgProvider: NewStdConfigProvider(&testCfg{Str: "old", Num: 0}),
					cfgSections: make(map[string]ConfigProvider),
				}
				app.cfgSections["section1"] = NewStdConfigProvider(&testSectionCfg{Enabled: false, Name: "old"})
				return app
			},
			setupFeeders: func() []Feeder {
				feeder := new(MockComplexFeeder)
				feeder.On("Feed", mock.Anything).Return(nil)
				// Due to map iteration order being random, either key could be processed first
				// If "section1" is processed first, it will fail and stop processing
				// If "_main" is processed first, it will succeed, then "section1" will fail
				feeder.On("FeedKey", "_main", mock.Anything).Return(nil).Maybe()
				feeder.On("FeedKey", "section1", mock.Anything).Return(ErrFeedKeyFailed)
				return []Feeder{feeder}
			},
			expectError: true,
			validateResult: func(t *testing.T, app *StdApplication) {
				// Configs should remain unchanged
				mainCfg := app.cfgProvider.GetConfig().(*testCfg)
				assert.Equal(t, "old", mainCfg.Str)

				sectionCfg := app.cfgSections["section1"].GetConfig().(*testSectionCfg)
				assert.False(t, sectionCfg.Enabled)
			},
		},
		{
			name: "non-pointer configs",
			setupApp: func() *StdApplication {
				mockLogger := new(MockLogger)
				mockLogger.On("Debug",
					"Creating new provider with updated config (original was non-pointer)",
					[]interface{}(nil)).Return()
				mockLogger.On("Debug", "Creating new provider for section", []interface{}{"section", "section1"}).Return()
				mockLogger.On("Debug", "Added main config for loading", mock.Anything).Return()
				mockLogger.On("Debug", "Added section config for loading", mock.Anything).Return()
				mockLogger.On("Debug", "Updated main config", mock.Anything).Return()
				mockLogger.On("Debug", "Updated section config", mock.Anything).Return()

				app := &StdApplication{
					logger:      mockLogger,
					cfgProvider: NewStdConfigProvider(testCfg{Str: "old", Num: 0}), // non-pointer
					cfgSections: make(map[string]ConfigProvider),
				}
				app.cfgSections["section1"] = NewStdConfigProvider(testSectionCfg{Enabled: false, Name: "old"}) // non-pointer
				return app
			},
			setupFeeders: func() []Feeder {
				feeder := new(MockComplexFeeder)
				feeder.On("Feed", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
					if cfg, ok := args.Get(0).(*testCfg); ok {
						cfg.Str = updatedValue
						cfg.Num = 42
					} else if cfg, ok := args.Get(0).(*testSectionCfg); ok {
						cfg.Enabled = true
						cfg.Name = "updated"
					}
				})
				feeder.On("FeedKey", "_main", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
					cfg := args.Get(1).(*testCfg)
					cfg.Str = updatedValue
					cfg.Num = 42
				})
				feeder.On("FeedKey", "section1", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
					cfg := args.Get(1).(*testSectionCfg)
					cfg.Enabled = true
					cfg.Name = "updated"
				})
				return []Feeder{feeder}
			},
			expectError: false,
			validateResult: func(t *testing.T, app *StdApplication) {
				mainCfg := app.cfgProvider.GetConfig()
				assert.Equal(t, updatedValue, mainCfg.(testCfg).Str)
				assert.Equal(t, 42, mainCfg.(testCfg).Num)

				sectionCfg := app.cfgSections["section1"].GetConfig()
				assert.True(t, sectionCfg.(testSectionCfg).Enabled)
				assert.Equal(t, "updated", sectionCfg.(testSectionCfg).Name)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := tt.setupApp()
			// Use per-app feeders; StdApplication exposes SetConfigFeeders directly.
			app.SetConfigFeeders(tt.setupFeeders())

			err := loadAppConfig(app)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				tt.validateResult(t, app)
			}

			// Assert that all mock expectations were met on the feeders we injected
			for _, feeder := range app.configFeeders {
				if mockFeeder, ok := feeder.(*MockComplexFeeder); ok {
					mockFeeder.AssertExpectations(t)
				}
			}
			if mockLogger, ok := app.logger.(*MockLogger); ok {
				mockLogger.AssertExpectations(t)
			}
		})
	}
}

// Mock for VerboseAwareFeeder
type MockVerboseAwareFeeder struct {
	mock.Mock
}

func (m *MockVerboseAwareFeeder) Feed(structure interface{}) error {
	args := m.Called(structure)
	if err := args.Error(0); err != nil {
		return fmt.Errorf("mock feeder error: %w", err)
	}
	return nil
}

func (m *MockVerboseAwareFeeder) SetVerboseDebug(enabled bool, logger interface{ Debug(msg string, args ...any) }) {
	m.Called(enabled, logger)
}

func TestConfig_SetVerboseDebug(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name               string
		setVerbose         bool
		feeders            []Feeder
		expectVerboseCalls int
	}{
		{
			name:       "enable verbose debug with verbose-aware feeder",
			setVerbose: true,
			feeders: []Feeder{
				&MockVerboseAwareFeeder{},
				&MockComplexFeeder{}, // non-verbose aware feeder
			},
			expectVerboseCalls: 1,
		},
		{
			name:       "disable verbose debug with verbose-aware feeder",
			setVerbose: false,
			feeders: []Feeder{
				&MockVerboseAwareFeeder{},
			},
			expectVerboseCalls: 1,
		},
		{
			name:       "enable verbose debug with no verbose-aware feeders",
			setVerbose: true,
			feeders: []Feeder{
				&MockComplexFeeder{},
			},
			expectVerboseCalls: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockLogger := new(MockLogger)

			// Set up the config with feeders already added (no verbose initially)
			cfg := NewConfig()
			for _, feeder := range tt.feeders {
				cfg.AddFeeder(feeder)
			}

			// Set up expectations for SetVerboseDebug call
			for _, feeder := range tt.feeders {
				if mockVerbose, ok := feeder.(*MockVerboseAwareFeeder); ok {
					mockVerbose.On("SetVerboseDebug", tt.setVerbose, mockLogger).Return()
				}
			}

			// Call SetVerboseDebug
			result := cfg.SetVerboseDebug(tt.setVerbose, mockLogger)

			// Assertions
			assert.Equal(t, cfg, result, "SetVerboseDebug should return the same config instance")
			assert.Equal(t, tt.setVerbose, cfg.VerboseDebug)
			assert.Equal(t, mockLogger, cfg.Logger)

			// Verify mock expectations
			for _, feeder := range tt.feeders {
				if mockVerbose, ok := feeder.(*MockVerboseAwareFeeder); ok {
					mockVerbose.AssertExpectations(t)
				}
			}
		})
	}
}

func TestConfig_AddFeeder_WithVerboseDebug(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name              string
		verboseEnabled    bool
		feeder            Feeder
		expectVerboseCall bool
	}{
		{
			name:              "add verbose-aware feeder with verbose enabled",
			verboseEnabled:    true,
			feeder:            &MockVerboseAwareFeeder{},
			expectVerboseCall: true,
		},
		{
			name:              "add verbose-aware feeder with verbose disabled",
			verboseEnabled:    false,
			feeder:            &MockVerboseAwareFeeder{},
			expectVerboseCall: false,
		},
		{
			name:              "add non-verbose-aware feeder",
			verboseEnabled:    true,
			feeder:            &MockComplexFeeder{},
			expectVerboseCall: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockLogger := new(MockLogger)

			cfg := NewConfig()
			cfg.VerboseDebug = tt.verboseEnabled
			cfg.Logger = mockLogger

			// Set up expectations for verbose-aware feeders
			if tt.expectVerboseCall {
				if mockVerbose, ok := tt.feeder.(*MockVerboseAwareFeeder); ok {
					mockVerbose.On("SetVerboseDebug", true, mockLogger).Return()
				}
			}

			// Call AddFeeder
			result := cfg.AddFeeder(tt.feeder)

			// Assertions
			assert.Equal(t, cfg, result, "AddFeeder should return the same config instance")
			assert.Contains(t, cfg.Feeders, tt.feeder)

			// Verify mock expectations
			if mockVerbose, ok := tt.feeder.(*MockVerboseAwareFeeder); ok {
				mockVerbose.AssertExpectations(t)
			}
		})
	}
}

func TestConfig_Feed_VerboseDebug(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		enableVerbose bool
	}{
		{
			name:          "verbose debug enabled",
			enableVerbose: true,
		},
		{
			name:          "verbose debug disabled",
			enableVerbose: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockLogger := new(MockLogger)

			cfg := NewConfig()
			if tt.enableVerbose {
				cfg.SetVerboseDebug(true, mockLogger)
				// Just allow any debug calls - we don't care about specific messages
				mockLogger.On("Debug", mock.Anything, mock.Anything).Return().Maybe()
			}

			cfg.AddStructKey("test", &testCfg{Str: "test", Num: 42})

			// Mock feeder that does nothing
			mockFeeder := new(MockComplexFeeder)
			mockFeeder.On("Feed", mock.Anything).Return(nil).Maybe()
			mockFeeder.On("FeedKey", mock.Anything, mock.Anything).Return(nil).Maybe()
			cfg.AddFeeder(mockFeeder)

			err := cfg.Feed()
			require.NoError(t, err)

			// Verify that verbose state was set correctly
			assert.Equal(t, tt.enableVerbose, cfg.VerboseDebug)
		})
	}
}

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

// TestDeepCopyValue_Maps tests deep copying of maps
func TestDeepCopyValue_Maps(t *testing.T) {
	t.Parallel()

	t.Run("simple map of strings", func(t *testing.T) {
		src := map[string]string{
			"key1": "value1",
			"key2": "value2",
		}

		dst := reflect.New(reflect.TypeOf(src)).Elem()
		deepCopyValue(dst, reflect.ValueOf(src))

		dstMap := dst.Interface().(map[string]string)
		assert.Equal(t, "value1", dstMap["key1"])
		assert.Equal(t, "value2", dstMap["key2"])

		// Verify it's a deep copy by modifying the source
		src["key1"] = "modified"
		assert.Equal(t, "value1", dstMap["key1"], "Destination should not be affected by source modification")
	})

	t.Run("map with string slice values", func(t *testing.T) {
		src := map[string][]string{
			"list1": {"a", "b", "c"},
			"list2": {"x", "y", "z"},
		}

		dst := reflect.New(reflect.TypeOf(src)).Elem()
		deepCopyValue(dst, reflect.ValueOf(src))

		dstMap := dst.Interface().(map[string][]string)
		assert.Equal(t, []string{"a", "b", "c"}, dstMap["list1"])
		assert.Equal(t, []string{"x", "y", "z"}, dstMap["list2"])

		// Verify it's a deep copy by modifying the source slice
		src["list1"][0] = "modified"
		assert.Equal(t, "a", dstMap["list1"][0], "Destination slice should not be affected")
	})

	t.Run("nested maps", func(t *testing.T) {
		src := map[string]map[string]int{
			"group1": {"a": 1, "b": 2},
			"group2": {"x": 10, "y": 20},
		}

		dst := reflect.New(reflect.TypeOf(src)).Elem()
		deepCopyValue(dst, reflect.ValueOf(src))

		dstMap := dst.Interface().(map[string]map[string]int)
		assert.Equal(t, 1, dstMap["group1"]["a"])
		assert.Equal(t, 20, dstMap["group2"]["y"])

		// Verify it's a deep copy
		src["group1"]["a"] = 999
		assert.Equal(t, 1, dstMap["group1"]["a"], "Nested map should not be affected")
	})

	t.Run("nil map", func(t *testing.T) {
		var src map[string]string = nil

		dst := reflect.New(reflect.TypeOf(map[string]string{})).Elem()
		deepCopyValue(dst, reflect.ValueOf(src))

		// For nil map, deepCopyValue returns early without modifying dst
		// dst remains as zero value which is nil for maps
		assert.True(t, !dst.IsValid() || dst.IsNil(), "Destination should remain nil for nil source")
	})
}

// TestDeepCopyValue_Slices tests deep copying of slices
func TestDeepCopyValue_Slices(t *testing.T) {
	t.Parallel()

	t.Run("simple slice of integers", func(t *testing.T) {
		src := []int{1, 2, 3, 4, 5}

		dst := reflect.New(reflect.TypeOf(src)).Elem()
		deepCopyValue(dst, reflect.ValueOf(src))

		dstSlice := dst.Interface().([]int)
		assert.Equal(t, []int{1, 2, 3, 4, 5}, dstSlice)

		// Verify it's a deep copy
		src[0] = 999
		assert.Equal(t, 1, dstSlice[0], "Destination should not be affected")
	})

	t.Run("slice of strings", func(t *testing.T) {
		src := []string{"hello", "world"}

		dst := reflect.New(reflect.TypeOf(src)).Elem()
		deepCopyValue(dst, reflect.ValueOf(src))

		dstSlice := dst.Interface().([]string)
		assert.Equal(t, []string{"hello", "world"}, dstSlice)

		src[0] = "modified"
		assert.Equal(t, "hello", dstSlice[0])
	})

	t.Run("slice of maps", func(t *testing.T) {
		src := []map[string]int{
			{"a": 1, "b": 2},
			{"x": 10, "y": 20},
		}

		dst := reflect.New(reflect.TypeOf(src)).Elem()
		deepCopyValue(dst, reflect.ValueOf(src))

		dstSlice := dst.Interface().([]map[string]int)
		assert.Equal(t, 1, dstSlice[0]["a"])
		assert.Equal(t, 20, dstSlice[1]["y"])

		// Verify it's a deep copy
		src[0]["a"] = 999
		assert.Equal(t, 1, dstSlice[0]["a"], "Nested map in slice should not be affected")
	})

	t.Run("nil slice", func(t *testing.T) {
		var src []string = nil

		dst := reflect.New(reflect.TypeOf([]string{})).Elem()
		deepCopyValue(dst, reflect.ValueOf(src))

		// For nil slice, deepCopyValue returns early without modifying dst
		// dst remains as zero value which is nil for slices
		assert.True(t, !dst.IsValid() || dst.IsNil(), "Destination should remain nil for nil source")
	})
}

// TestDeepCopyValue_Pointers tests deep copying of pointers
func TestDeepCopyValue_Pointers(t *testing.T) {
	t.Parallel()

	t.Run("pointer to string", func(t *testing.T) {
		str := "original"
		src := &str

		dst := reflect.New(reflect.TypeOf(src)).Elem()
		deepCopyValue(dst, reflect.ValueOf(src))

		dstPtr := dst.Interface().(*string)
		assert.Equal(t, "original", *dstPtr)

		// Verify it's a deep copy
		*src = "modified"
		assert.Equal(t, "original", *dstPtr, "Destination should not be affected")
	})

	t.Run("pointer to struct", func(t *testing.T) {
		type TestStruct struct {
			Name  string
			Value int
		}

		src := &TestStruct{Name: "test", Value: 42}

		dst := reflect.New(reflect.TypeOf(src)).Elem()
		deepCopyValue(dst, reflect.ValueOf(src))

		dstPtr := dst.Interface().(*TestStruct)
		assert.Equal(t, "test", dstPtr.Name)
		assert.Equal(t, 42, dstPtr.Value)

		// Verify it's a deep copy
		src.Name = "modified"
		assert.Equal(t, "test", dstPtr.Name)
	})

	t.Run("nil pointer", func(t *testing.T) {
		var src *string = nil

		dst := reflect.New(reflect.TypeOf((*string)(nil))).Elem()
		deepCopyValue(dst, reflect.ValueOf(src))

		// For nil pointer, deepCopyValue returns early without modifying dst
		// dst remains as zero value which is nil for pointers
		assert.True(t, !dst.IsValid() || dst.IsNil(), "Destination should remain nil for nil source")
	})
}

// TestDeepCopyValue_Structs tests deep copying of structs
func TestDeepCopyValue_Structs(t *testing.T) {
	t.Parallel()

	t.Run("simple struct", func(t *testing.T) {
		type SimpleStruct struct {
			Name string
			Age  int
		}

		src := SimpleStruct{Name: "John", Age: 30}

		dst := reflect.New(reflect.TypeOf(src)).Elem()
		deepCopyValue(dst, reflect.ValueOf(src))

		dstStruct := dst.Interface().(SimpleStruct)
		assert.Equal(t, "John", dstStruct.Name)
		assert.Equal(t, 30, dstStruct.Age)
	})

	t.Run("struct with map field", func(t *testing.T) {
		type ConfigStruct struct {
			Name     string
			Settings map[string]string
		}

		src := ConfigStruct{
			Name:     "config1",
			Settings: map[string]string{"key1": "value1", "key2": "value2"},
		}

		dst := reflect.New(reflect.TypeOf(src)).Elem()
		deepCopyValue(dst, reflect.ValueOf(src))

		dstStruct := dst.Interface().(ConfigStruct)
		assert.Equal(t, "config1", dstStruct.Name)
		assert.Equal(t, "value1", dstStruct.Settings["key1"])

		// Verify it's a deep copy - THIS TESTS THE KEY BUG FIX
		src.Settings["key1"] = "modified"
		assert.Equal(t, "value1", dstStruct.Settings["key1"], "Map in struct should not be affected")
	})

	t.Run("struct with slice field", func(t *testing.T) {
		type ListStruct struct {
			Name  string
			Items []string
		}

		src := ListStruct{
			Name:  "list1",
			Items: []string{"a", "b", "c"},
		}

		dst := reflect.New(reflect.TypeOf(src)).Elem()
		deepCopyValue(dst, reflect.ValueOf(src))

		dstStruct := dst.Interface().(ListStruct)
		assert.Equal(t, "list1", dstStruct.Name)
		assert.Equal(t, []string{"a", "b", "c"}, dstStruct.Items)

		// Verify it's a deep copy
		src.Items[0] = "modified"
		assert.Equal(t, "a", dstStruct.Items[0], "Slice in struct should not be affected")
	})

	t.Run("nested struct", func(t *testing.T) {
		type InnerStruct struct {
			Value int
		}
		type OuterStruct struct {
			Name  string
			Inner InnerStruct
		}

		src := OuterStruct{
			Name:  "outer",
			Inner: InnerStruct{Value: 42},
		}

		dst := reflect.New(reflect.TypeOf(src)).Elem()
		deepCopyValue(dst, reflect.ValueOf(src))

		dstStruct := dst.Interface().(OuterStruct)
		assert.Equal(t, "outer", dstStruct.Name)
		assert.Equal(t, 42, dstStruct.Inner.Value)
	})

	t.Run("struct with unexported fields", func(t *testing.T) {
		type StructWithPrivate struct {
			Public  string
			private int
		}

		src := StructWithPrivate{Public: "visible", private: 42}

		dst := reflect.New(reflect.TypeOf(src)).Elem()
		// Should not panic even with unexported fields
		require.NotPanics(t, func() {
			deepCopyValue(dst, reflect.ValueOf(src))
		})

		dstStruct := dst.Interface().(StructWithPrivate)
		assert.Equal(t, "visible", dstStruct.Public)
	})
}

// TestDeepCopyValue_BasicTypes tests deep copying of basic types
func TestDeepCopyValue_BasicTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value interface{}
	}{
		{"int", 42},
		{"int64", int64(123456789)},
		{"float64", 3.14159},
		{"string", "hello world"},
		{"bool", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := reflect.ValueOf(tt.value)
			dst := reflect.New(src.Type()).Elem()

			deepCopyValue(dst, src)

			assert.Equal(t, tt.value, dst.Interface())
		})
	}
}

// TestDeepCopyValue_ComplexStructures tests deep copying of complex nested structures
func TestDeepCopyValue_ComplexStructures(t *testing.T) {
	t.Parallel()

	type ComplexConfig struct {
		Name            string
		BackendServices map[string]string
		Features        map[string]bool
		AllowedIPs      []string
	}

	src := ComplexConfig{
		Name: "tenant1",
		BackendServices: map[string]string{
			"api":    "https://api.example.com",
			"legacy": "https://legacy.example.com",
		},
		Features: map[string]bool{
			"feature1": true,
			"feature2": false,
		},
		AllowedIPs: []string{"192.168.1.1", "10.0.0.1"},
	}

	dst := reflect.New(reflect.TypeOf(src)).Elem()
	deepCopyValue(dst, reflect.ValueOf(src))

	dstConfig := dst.Interface().(ComplexConfig)

	// Verify all fields copied correctly
	assert.Equal(t, "tenant1", dstConfig.Name)
	assert.Equal(t, "https://api.example.com", dstConfig.BackendServices["api"])
	assert.Equal(t, "https://legacy.example.com", dstConfig.BackendServices["legacy"])
	assert.True(t, dstConfig.Features["feature1"])
	assert.False(t, dstConfig.Features["feature2"])
	assert.Equal(t, []string{"192.168.1.1", "10.0.0.1"}, dstConfig.AllowedIPs)

	// Verify deep copy by modifying source
	src.BackendServices["api"] = "https://modified.example.com"
	src.Features["feature1"] = false
	src.AllowedIPs[0] = "1.1.1.1"

	// Destination should NOT be affected (isolation is preserved)
	assert.Equal(t, "https://api.example.com", dstConfig.BackendServices["api"], "BackendServices map should be deep copied")
	assert.True(t, dstConfig.Features["feature1"], "Features map should be deep copied")
	assert.Equal(t, "192.168.1.1", dstConfig.AllowedIPs[0], "AllowedIPs slice should be deep copied")
}

// TestDeepCopyValue_Arrays tests deep copying of fixed-size arrays
func TestDeepCopyValue_Arrays(t *testing.T) {
	t.Parallel()

	t.Run("array of integers", func(t *testing.T) {
		src := [5]int{1, 2, 3, 4, 5}

		dst := reflect.New(reflect.TypeOf(src)).Elem()
		deepCopyValue(dst, reflect.ValueOf(src))

		dstArray := dst.Interface().([5]int)
		assert.Equal(t, [5]int{1, 2, 3, 4, 5}, dstArray)

		// Arrays are value types in Go, but let's verify the copy works
		src[0] = 999
		assert.Equal(t, 1, dstArray[0], "Destination array should not be affected")
	})

	t.Run("array of strings", func(t *testing.T) {
		src := [3]string{"foo", "bar", "baz"}

		dst := reflect.New(reflect.TypeOf(src)).Elem()
		deepCopyValue(dst, reflect.ValueOf(src))

		dstArray := dst.Interface().([3]string)
		assert.Equal(t, [3]string{"foo", "bar", "baz"}, dstArray)

		src[1] = "modified"
		assert.Equal(t, "bar", dstArray[1])
	})

	t.Run("array of pointers", func(t *testing.T) {
		str1, str2 := "value1", "value2"
		src := [2]*string{&str1, &str2}

		dst := reflect.New(reflect.TypeOf(src)).Elem()
		deepCopyValue(dst, reflect.ValueOf(src))

		dstArray := dst.Interface().([2]*string)
		assert.Equal(t, "value1", *dstArray[0])
		assert.Equal(t, "value2", *dstArray[1])

		// Verify deep copy - modifying source pointer values shouldn't affect destination
		*src[0] = "modified"
		assert.Equal(t, "value1", *dstArray[0], "Array of pointers should be deep copied")
	})
}

// TestDeepCopyValue_Interfaces tests deep copying of interface values
func TestDeepCopyValue_Interfaces(t *testing.T) {
	t.Parallel()

	t.Run("interface with concrete string", func(t *testing.T) {
		var src interface{} = "hello"

		dst := reflect.New(reflect.TypeOf(src)).Elem()
		deepCopyValue(dst, reflect.ValueOf(src))

		dstValue := dst.Interface()
		assert.Equal(t, "hello", dstValue)
	})

	t.Run("interface with concrete map", func(t *testing.T) {
		var src interface{} = map[string]int{"a": 1, "b": 2}

		dst := reflect.New(reflect.TypeOf(src)).Elem()
		deepCopyValue(dst, reflect.ValueOf(src))

		dstValue := dst.Interface().(map[string]int)
		assert.Equal(t, 1, dstValue["a"])
		assert.Equal(t, 2, dstValue["b"])

		// Verify it's a deep copy
		srcMap := src.(map[string]int)
		srcMap["a"] = 999
		assert.Equal(t, 1, dstValue["a"], "Interface containing map should be deep copied")
	})

	t.Run("struct with interface field", func(t *testing.T) {
		type ConfigWithInterface struct {
			Name string
			Data interface{}
		}

		src := ConfigWithInterface{
			Name: "test",
			Data: map[string]int{"count": 42},
		}

		dst := reflect.New(reflect.TypeOf(src)).Elem()
		deepCopyValue(dst, reflect.ValueOf(src))

		dstValue := dst.Interface().(ConfigWithInterface)
		assert.Equal(t, "test", dstValue.Name)
		assert.Equal(t, 42, dstValue.Data.(map[string]int)["count"])

		// Verify deep copy
		srcMap := src.Data.(map[string]int)
		srcMap["count"] = 999
		assert.Equal(t, 42, dstValue.Data.(map[string]int)["count"], "Interface field containing map should be deep copied")
	})

	t.Run("struct with nil interface field", func(t *testing.T) {
		type ConfigWithInterface struct {
			Name string
			Data interface{}
		}

		src := ConfigWithInterface{
			Name: "test",
			Data: nil,
		}

		dst := reflect.New(reflect.TypeOf(src)).Elem()
		deepCopyValue(dst, reflect.ValueOf(src))

		dstValue := dst.Interface().(ConfigWithInterface)
		assert.Equal(t, "test", dstValue.Name)
		assert.Nil(t, dstValue.Data, "Nil interface field should remain nil")
	})

	t.Run("interface with struct", func(t *testing.T) {
		type TestStruct struct {
			Value int
			Data  map[string]string
		}

		var src interface{} = TestStruct{
			Value: 42,
			Data:  map[string]string{"key": "value"},
		}

		dst := reflect.New(reflect.TypeOf(src)).Elem()
		deepCopyValue(dst, reflect.ValueOf(src))

		dstValue := dst.Interface().(TestStruct)
		assert.Equal(t, 42, dstValue.Value)
		assert.Equal(t, "value", dstValue.Data["key"])

		// Verify deep copy of the map inside the struct
		srcStruct := src.(TestStruct)
		srcStruct.Data["key"] = "modified"
		assert.Equal(t, "value", dstValue.Data["key"], "Interface with struct containing map should be deep copied")
	})
}

// TestDeepCopyValue_Channels tests copying of channels (by reference)
func TestDeepCopyValue_Channels(t *testing.T) {
	t.Parallel()

	t.Run("channel of integers", func(t *testing.T) {
		src := make(chan int, 2)
		src <- 42
		src <- 100

		dst := reflect.New(reflect.TypeOf(src)).Elem()
		deepCopyValue(dst, reflect.ValueOf(src))

		dstChan := dst.Interface().(chan int)

		// Channels are copied by reference, so they should be the same channel
		assert.Equal(t, 42, <-dstChan, "Channel should be copied by reference")
		assert.Equal(t, 100, <-dstChan, "Channel should be copied by reference")
	})

	t.Run("nil channel", func(t *testing.T) {
		var src chan string = nil

		dst := reflect.New(reflect.TypeOf(src)).Elem()
		deepCopyValue(dst, reflect.ValueOf(src))

		dstChan := dst.Interface().(chan string)
		assert.Nil(t, dstChan, "Nil channel should remain nil")
	})
}

// TestDeepCopyValue_Functions tests copying of functions (by reference)
func TestDeepCopyValue_Functions(t *testing.T) {
	t.Parallel()

	t.Run("function value", func(t *testing.T) {
		callCount := 0
		src := func(x int) int {
			callCount++
			return x * 2
		}

		dst := reflect.New(reflect.TypeOf(src)).Elem()
		deepCopyValue(dst, reflect.ValueOf(src))

		dstFunc := dst.Interface().(func(int) int)

		// Functions are copied by reference, so calling either increments the same counter
		assert.Equal(t, 10, dstFunc(5))
		assert.Equal(t, 1, callCount, "Function should be copied by reference")

		assert.Equal(t, 20, src(10))
		assert.Equal(t, 2, callCount, "Both function references share state")
	})

	t.Run("nil function", func(t *testing.T) {
		var src func(int) int = nil

		dst := reflect.New(reflect.TypeOf(src)).Elem()
		deepCopyValue(dst, reflect.ValueOf(src))

		dstFunc := dst.Interface().(func(int) int)
		assert.Nil(t, dstFunc, "Nil function should remain nil")
	})
}

// TestDeepCopyValue_Invalid tests handling of invalid reflect values
func TestDeepCopyValue_Invalid(t *testing.T) {
	t.Parallel()

	t.Run("invalid value", func(t *testing.T) {
		var src reflect.Value // Invalid (zero value)

		dst := reflect.New(reflect.TypeOf("")).Elem()

		// Should not panic
		require.NotPanics(t, func() {
			deepCopyValue(dst, src)
		})
	})
}
