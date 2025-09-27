package cache

import (
	"context"
	"errors"
)

// Multi-operations BDD test steps (SetMulti, GetMulti, DeleteMulti, Flush)

func (ctx *CacheBDDTestContext) iHaveSetMultipleCacheItems() error {
	items := map[string]interface{}{
		"item1": "value1",
		"item2": "value2",
		"item3": "value3",
	}

	for key, value := range items {
		err := ctx.service.Set(context.Background(), key, value, 0)
		if err != nil {
			return err
		}
	}

	ctx.multipleItems = items
	return nil
}

func (ctx *CacheBDDTestContext) iFlushAllCacheItems() error {
	err := ctx.service.Flush(context.Background())
	if err != nil {
		ctx.lastError = err
	}
	return err
}

func (ctx *CacheBDDTestContext) iGetAnyOfThePreviouslySetCacheItems() error {
	// Try to get any item from the previously set items
	for key := range ctx.multipleItems {
		value, found := ctx.service.Get(context.Background(), key)
		ctx.cachedValue = value
		ctx.cacheHit = found
		break
	}
	return nil
}

func (ctx *CacheBDDTestContext) iSetMultipleCacheItemsWithDifferentKeysAndValues() error {
	items := map[string]interface{}{
		"multi-key1": "multi-value1",
		"multi-key2": "multi-value2",
		"multi-key3": "multi-value3",
	}

	err := ctx.service.SetMulti(context.Background(), items, 0)
	if err != nil {
		ctx.lastError = err
		return err
	}

	ctx.multipleItems = items
	return nil
}

func (ctx *CacheBDDTestContext) allItemsShouldBeStoredSuccessfully() error {
	if ctx.lastError != nil {
		return ctx.lastError
	}
	return nil
}

func (ctx *CacheBDDTestContext) iShouldBeAbleToRetrieveAllItems() error {
	for key, expectedValue := range ctx.multipleItems {
		value, found := ctx.service.Get(context.Background(), key)
		if !found {
			return errors.New("item should be found in cache")
		}
		if value != expectedValue {
			return errors.New("cached value does not match expected value")
		}
	}
	return nil
}

func (ctx *CacheBDDTestContext) iHaveSetMultipleCacheItemsWithKeys(key1, key2, key3 string) error {
	items := map[string]interface{}{
		key1: "value1",
		key2: "value2",
		key3: "value3",
	}

	for key, value := range items {
		err := ctx.service.Set(context.Background(), key, value, 0)
		if err != nil {
			return err
		}
	}

	ctx.multipleItems = items
	return nil
}

func (ctx *CacheBDDTestContext) iGetMultipleCacheItemsWithTheSameKeys() error {
	// Get keys from the stored items
	keys := make([]string, 0, len(ctx.multipleItems))
	for key := range ctx.multipleItems {
		keys = append(keys, key)
	}

	result, err := ctx.service.GetMulti(context.Background(), keys)
	if err != nil {
		ctx.lastError = err
		return err
	}

	ctx.multipleResult = result
	return nil
}

func (ctx *CacheBDDTestContext) iShouldReceiveAllTheCachedValues() error {
	if len(ctx.multipleResult) != len(ctx.multipleItems) {
		return errors.New("should receive all cached values")
	}
	return nil
}

func (ctx *CacheBDDTestContext) theValuesShouldMatchWhatWasStored() error {
	for key, expectedValue := range ctx.multipleItems {
		actualValue, found := ctx.multipleResult[key]
		if !found {
			return errors.New("value should be found in results")
		}
		if actualValue != expectedValue {
			return errors.New("value does not match what was stored")
		}
	}
	return nil
}

func (ctx *CacheBDDTestContext) iHaveSetMultipleCacheItemsWithKeysForDeletion(key1, key2, key3 string) error {
	items := map[string]interface{}{
		key1: "value1",
		key2: "value2",
		key3: "value3",
	}

	for key, value := range items {
		err := ctx.service.Set(context.Background(), key, value, 0)
		if err != nil {
			return err
		}
	}

	ctx.multipleItems = items
	return nil
}

func (ctx *CacheBDDTestContext) iDeleteMultipleCacheItemsWithTheSameKeys() error {
	// Get keys from the stored items
	keys := make([]string, 0, len(ctx.multipleItems))
	for key := range ctx.multipleItems {
		keys = append(keys, key)
	}

	err := ctx.service.DeleteMulti(context.Background(), keys)
	if err != nil {
		ctx.lastError = err
		return err
	}
	return nil
}

func (ctx *CacheBDDTestContext) iShouldReceiveNoCachedValues() error {
	if len(ctx.multipleResult) != 0 {
		return errors.New("should receive no cached values")
	}
	return nil
}
