package feeders

import "reflect"

// TenantAffixedEnvFeeder is a feeder that reads environment variables with tenant-specific prefixes and suffixes
type TenantAffixedEnvFeeder struct {
	*AffixedEnvFeeder
	SetPrefixFunc func(string)
	SetSuffixFunc func(string)
	verboseDebug  bool
	logger        interface {
		Debug(msg string, args ...any)
	}
}

// NewTenantAffixedEnvFeeder creates a new TenantAffixedEnvFeeder with the given prefix and suffix functions
// The prefix and suffix functions are used to modify the prefix and suffix of the environment variables
// before they are used to set the struct fields
// The prefix function is used to modify the prefix of the environment variables
// The suffix function is used to modify the suffix of the environment variables
func NewTenantAffixedEnvFeeder(prefix, suffix func(string) string) *TenantAffixedEnvFeeder {
	affixedFeeder := NewAffixedEnvFeeder("", "") // Initialize with empty prefix and suffix (already returns pointer)
	result := TenantAffixedEnvFeeder{
		AffixedEnvFeeder: affixedFeeder, // Already a pointer
		verboseDebug:     false,
		logger:           nil,
	}

	// Set the function closures to modify the affixed feeder
	result.SetPrefixFunc = func(p string) {
		result.Prefix = prefix(p)
	}
	result.SetSuffixFunc = func(s string) {
		result.Suffix = suffix(s)
	}

	return &result
}

// Feed implements the basic Feeder interface but requires tenant context
// For TenantAffixedEnvFeeder, use FeedKey instead to provide tenant context
// This method succeeds without doing anything if called without tenant context,
// since FeedKey will be called next by ComplexFeeder-aware config loaders
func (f *TenantAffixedEnvFeeder) Feed(structure interface{}) error {
	if f.verboseDebug && f.logger != nil {
		f.logger.Debug("TenantAffixedEnvFeeder: Feed called without tenant context, checking if prefix/suffix are set")
	}

	// If prefix and suffix have been set (via SetPrefixFunc/SetSuffixFunc), use them
	if f.AffixedEnvFeeder != nil && (f.Prefix != "" || f.Suffix != "") {
		if f.verboseDebug && f.logger != nil {
			f.logger.Debug("TenantAffixedEnvFeeder: Using pre-configured prefix/suffix", "prefix", f.Prefix, "suffix", f.Suffix)
		}
		return f.AffixedEnvFeeder.Feed(structure)
	}

	// No prefix/suffix set - succeed gracefully, FeedKey will be called next
	if f.verboseDebug && f.logger != nil {
		f.logger.Debug("TenantAffixedEnvFeeder: No prefix/suffix set, skipping Feed (FeedKey will be called next)")
	}
	return nil
}

// SetVerboseDebug enables or disables verbose debug logging
func (f *TenantAffixedEnvFeeder) SetVerboseDebug(enabled bool, logger interface{ Debug(msg string, args ...any) }) {
	f.verboseDebug = enabled
	f.logger = logger
	// Also enable verbose debug on the underlying AffixedEnvFeeder
	if f.AffixedEnvFeeder != nil {
		f.AffixedEnvFeeder.SetVerboseDebug(enabled, logger)
	}
	if enabled && logger != nil {
		f.logger.Debug("Verbose tenant affixed environment feeder debugging enabled")
	}
}

// SetFieldTracker sets the field tracker for recording field populations
func (f *TenantAffixedEnvFeeder) SetFieldTracker(tracker FieldTracker) {
	// Delegate to the embedded AffixedEnvFeeder
	if f.AffixedEnvFeeder != nil {
		f.AffixedEnvFeeder.SetFieldTracker(tracker)
	}
}

// FeedKey implements the ComplexFeeder interface for tenant-specific feeding
// Note: When called from tenant config loader, prefix/suffix are already configured
// The key parameter here is actually the config section name, not tenant ID
func (f *TenantAffixedEnvFeeder) FeedKey(tenantID string, structure interface{}) error {
	if f.verboseDebug && f.logger != nil {
		f.logger.Debug("TenantAffixedEnvFeeder: FeedKey called", "key", tenantID, "structureType", reflect.TypeOf(structure), "currentPrefix", f.Prefix, "currentSuffix", f.Suffix)
	}

	// If prefix or suffix are already set, use them (tenant loader pre-configured them)
	// Otherwise, set them using the key parameter (for backward compatibility)
	if f.Prefix == "" && f.Suffix == "" {
		// Set tenant-specific prefix and suffix using the provided functions
		if f.SetPrefixFunc != nil {
			f.SetPrefixFunc(tenantID)
			if f.verboseDebug && f.logger != nil {
				f.logger.Debug("TenantAffixedEnvFeeder: Set prefix from key", "key", tenantID, "prefix", f.Prefix)
			}
		}

		if f.SetSuffixFunc != nil {
			f.SetSuffixFunc(tenantID)
			if f.verboseDebug && f.logger != nil {
				f.logger.Debug("TenantAffixedEnvFeeder: Set suffix from key", "key", tenantID, "suffix", f.Suffix)
			}
		}
	} else if f.verboseDebug && f.logger != nil {
		f.logger.Debug("TenantAffixedEnvFeeder: Using pre-configured prefix/suffix", "prefix", f.Prefix, "suffix", f.Suffix)
	}

	// Now call the underlying Feed method with the configured prefix/suffix
	err := f.AffixedEnvFeeder.Feed(structure)

	if f.verboseDebug && f.logger != nil {
		if err != nil {
			f.logger.Debug("TenantAffixedEnvFeeder: FeedKey completed with error", "key", tenantID, "error", err)
		} else {
			f.logger.Debug("TenantAffixedEnvFeeder: FeedKey completed successfully", "key", tenantID)
		}
	}

	return err
}
