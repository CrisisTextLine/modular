package cache

import (
	"testing"

	"github.com/cucumber/godog"
)

// Test runner function - Main BDD test registration
func TestCacheModuleBDD(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: func(ctx *godog.ScenarioContext) {
			testCtx := &CacheBDDTestContext{}

			// Background
			ctx.Step(`^I have a modular application with cache module configured$`, testCtx.iHaveAModularApplicationWithCacheModuleConfigured)

			// Initialization steps
			ctx.Step(`^the cache module is initialized$`, testCtx.theCacheModuleIsInitialized)
			ctx.Step(`^the cache service should be available$`, testCtx.theCacheServiceShouldBeAvailable)

			// Configuration steps
			ctx.Step(`^I have a cache configuration with memory engine$`, testCtx.iHaveACacheConfigurationWithMemoryEngine)
			ctx.Step(`^I have a cache configuration with redis engine$`, testCtx.iHaveACacheConfigurationWithRedisEngine)
			ctx.Step(`^the memory cache engine should be configured$`, testCtx.theMemoryCacheEngineShouldBeConfigured)
			ctx.Step(`^the redis cache engine should be configured$`, testCtx.theRedisCacheEngineShouldBeConfigured)
			ctx.Step(`^I have a cache configuration with invalid Redis settings$`, testCtx.iHaveACacheConfigurationWithInvalidRedisSettings)
			ctx.Step(`^the cache module attempts to start$`, testCtx.theCacheModuleAttemptsToStart)

			// Service availability
			ctx.Step(`^I have a cache service available$`, testCtx.iHaveACacheServiceAvailable)
			ctx.Step(`^I have a cache service with default TTL configured$`, testCtx.iHaveACacheServiceWithDefaultTTLConfigured)

			// Basic cache operations
			ctx.Step(`^I set a cache item with key "([^"]*)" and value "([^"]*)"$`, testCtx.iSetACacheItemWithKeyAndValue)
			ctx.Step(`^I get the cache item with key "([^"]*)"$`, testCtx.iGetTheCacheItemWithKey)
			ctx.Step(`^I get the cache item with key "([^"]*)" immediately$`, testCtx.iGetTheCacheItemWithKeyImmediately)
			ctx.Step(`^the cached value should be "([^"]*)"$`, testCtx.theCachedValueShouldBe)
			ctx.Step(`^the cache hit should be successful$`, testCtx.theCacheHitShouldBeSuccessful)
			ctx.Step(`^the cache hit should be unsuccessful$`, testCtx.theCacheHitShouldBeUnsuccessful)
			ctx.Step(`^no value should be returned$`, testCtx.noValueShouldBeReturned)

			// TTL operations
			ctx.Step(`^I set a cache item with key "([^"]*)" and value "([^"]*)" with TTL (\d+) seconds$`, testCtx.iSetACacheItemWithKeyAndValueWithTTLSeconds)
			ctx.Step(`^I wait for (\d+) seconds$`, testCtx.iWaitForSeconds)
			ctx.Step(`^I set a cache item without specifying TTL$`, testCtx.iSetACacheItemWithoutSpecifyingTTL)
			ctx.Step(`^the item should use the default TTL from configuration$`, testCtx.theItemShouldUseTheDefaultTTLFromConfiguration)

			// Delete operations
			ctx.Step(`^I have set a cache item with key "([^"]*)" and value "([^"]*)"$`, testCtx.iHaveSetACacheItemWithKeyAndValue)
			ctx.Step(`^I delete the cache item with key "([^"]*)"$`, testCtx.iDeleteTheCacheItemWithKey)

			// Flush operations
			ctx.Step(`^I have set multiple cache items$`, testCtx.iHaveSetMultipleCacheItems)
			ctx.Step(`^I flush all cache items$`, testCtx.iFlushAllCacheItems)
			ctx.Step(`^I get any of the previously set cache items$`, testCtx.iGetAnyOfThePreviouslySetCacheItems)

			// Multi operations
			ctx.Step(`^I set multiple cache items with different keys and values$`, testCtx.iSetMultipleCacheItemsWithDifferentKeysAndValues)
			ctx.Step(`^all items should be stored successfully$`, testCtx.allItemsShouldBeStoredSuccessfully)
			ctx.Step(`^I should be able to retrieve all items$`, testCtx.iShouldBeAbleToRetrieveAllItems)

			ctx.Step(`^I have set multiple cache items with keys "([^"]*)", "([^"]*)", "([^"]*)"$`, testCtx.iHaveSetMultipleCacheItemsWithKeys)
			ctx.Step(`^I get multiple cache items with the same keys$`, testCtx.iGetMultipleCacheItemsWithTheSameKeys)
			ctx.Step(`^I should receive all the cached values$`, testCtx.iShouldReceiveAllTheCachedValues)
			ctx.Step(`^the values should match what was stored$`, testCtx.theValuesShouldMatchWhatWasStored)
			ctx.Step(`^I have set multiple cache items with keys "([^"]*)", "([^"]*)", "([^"]*)" for deletion$`, testCtx.iHaveSetMultipleCacheItemsWithKeysForDeletion)
			ctx.Step(`^I delete multiple cache items with the same keys$`, testCtx.iDeleteMultipleCacheItemsWithTheSameKeys)
			ctx.Step(`^I should receive no cached values$`, testCtx.iShouldReceiveNoCachedValues)

			// Error handling
			ctx.Step(`^the module should handle connection errors gracefully$`, testCtx.theModuleShouldHandleConnectionErrorsGracefully)
			ctx.Step(`^appropriate error messages should be logged$`, testCtx.appropriateErrorMessagesShouldBeLogged)

			// Event observation steps
			ctx.Step(`^I have a cache service with event observation enabled$`, testCtx.iHaveACacheServiceWithEventObservationEnabled)
			ctx.Step(`^a cache set event should be emitted$`, testCtx.aCacheSetEventShouldBeEmitted)
			ctx.Step(`^the event should contain the cache key "([^"]*)"$`, testCtx.theEventShouldContainTheCacheKey)
			ctx.Step(`^a cache hit event should be emitted$`, testCtx.aCacheHitEventShouldBeEmitted)
			ctx.Step(`^a cache miss event should be emitted$`, testCtx.aCacheMissEventShouldBeEmitted)
			ctx.Step(`^I get a non-existent key "([^"]*)"$`, testCtx.iGetANonExistentKey)
			ctx.Step(`^a cache delete event should be emitted$`, testCtx.aCacheDeleteEventShouldBeEmitted)
			ctx.Step(`^the cache module starts$`, testCtx.theCacheModuleStarts)
			ctx.Step(`^a cache connected event should be emitted$`, testCtx.aCacheConnectedEventShouldBeEmitted)
			ctx.Step(`^a cache flush event should be emitted$`, testCtx.aCacheFlushEventShouldBeEmitted)
			ctx.Step(`^the cache module stops$`, testCtx.theCacheModuleStops)
			ctx.Step(`^a cache disconnected event should be emitted$`, testCtx.aCacheDisconnectedEventShouldBeEmitted)

			// Error event steps
			ctx.Step(`^the cache engine encounters a connection error$`, testCtx.theCacheEngineEncountersAConnectionError)
			ctx.Step(`^I attempt to start the cache module$`, testCtx.iAttemptToStartTheCacheModule)
			ctx.Step(`^a cache error event should be emitted$`, testCtx.aCacheErrorEventShouldBeEmitted)
			ctx.Step(`^the error event should contain connection error details$`, testCtx.theErrorEventShouldContainConnectionErrorDetails)

			// Expired event steps
			ctx.Step(`^the cache cleanup process runs$`, testCtx.theCacheCleanupProcessRuns)
			ctx.Step(`^a cache expired event should be emitted$`, testCtx.aCacheExpiredEventShouldBeEmitted)
			ctx.Step(`^the expired event should contain the expired key "([^"]*)"$`, testCtx.theExpiredEventShouldContainTheExpiredKey)

			// Evicted event steps
			ctx.Step(`^I have a cache service with small memory limit configured$`, testCtx.iHaveACacheServiceWithSmallMemoryLimitConfigured)
			ctx.Step(`^I have event observation enabled$`, testCtx.iHaveEventObservationEnabled)
			ctx.Step(`^I fill the cache beyond its maximum capacity$`, testCtx.iFillTheCacheBeyondItsMaximumCapacity)
			ctx.Step(`^a cache evicted event should be emitted$`, testCtx.aCacheEvictedEventShouldBeEmitted)
			ctx.Step(`^the evicted event should contain eviction details$`, testCtx.theEvictedEventShouldContainEvictionDetails)

			// Event validation (mega-scenario)
			ctx.Step(`^all registered events should be emitted during testing$`, testCtx.allRegisteredEventsShouldBeEmittedDuringTesting)
		},
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"features"},
			TestingT: t,
			Strict:   true,
		},
	}

	if suite.Run() != 0 {
		t.Fatal("non-zero status returned, failed to run feature tests")
	}
}
