package cache

import (
	"context"
	"errors"
	"time"
)

// Basic cache operations BDD test steps

func (ctx *CacheBDDTestContext) iSetACacheItemWithKeyAndValue(key, value string) error {
	err := ctx.service.Set(context.Background(), key, value, 0)
	if err != nil {
		ctx.lastError = err
	}
	return err
}

func (ctx *CacheBDDTestContext) iGetTheCacheItemWithKey(key string) error {
	value, found := ctx.service.Get(context.Background(), key)
	ctx.cachedValue = value
	ctx.cacheHit = found
	return nil
}

func (ctx *CacheBDDTestContext) theCachedValueShouldBe(expectedValue string) error {
	if !ctx.cacheHit {
		return errors.New("cache miss when hit was expected")
	}

	if ctx.cachedValue != expectedValue {
		return errors.New("cached value does not match expected value")
	}

	return nil
}

func (ctx *CacheBDDTestContext) theCacheHitShouldBeSuccessful() error {
	if !ctx.cacheHit {
		return errors.New("cache hit should have been successful")
	}
	return nil
}

func (ctx *CacheBDDTestContext) iSetACacheItemWithKeyAndValueWithTTLSeconds(key, value string, ttl int) error {
	duration := time.Duration(ttl) * time.Second
	err := ctx.service.Set(context.Background(), key, value, duration)
	if err != nil {
		ctx.lastError = err
	}
	return err
}

func (ctx *CacheBDDTestContext) iGetTheCacheItemWithKeyImmediately(key string) error {
	return ctx.iGetTheCacheItemWithKey(key)
}

func (ctx *CacheBDDTestContext) iWaitForSeconds(seconds int) error {
	time.Sleep(time.Duration(seconds) * time.Second)
	return nil
}

func (ctx *CacheBDDTestContext) theCacheHitShouldBeUnsuccessful() error {
	if ctx.cacheHit {
		return errors.New("cache hit should have been unsuccessful")
	}
	return nil
}

func (ctx *CacheBDDTestContext) noValueShouldBeReturned() error {
	if ctx.cachedValue != nil {
		return errors.New("no value should have been returned")
	}
	return nil
}

func (ctx *CacheBDDTestContext) iHaveSetACacheItemWithKeyAndValue(key, value string) error {
	return ctx.iSetACacheItemWithKeyAndValue(key, value)
}

func (ctx *CacheBDDTestContext) iDeleteTheCacheItemWithKey(key string) error {
	err := ctx.service.Delete(context.Background(), key)
	if err != nil {
		ctx.lastError = err
	}
	return err
}
