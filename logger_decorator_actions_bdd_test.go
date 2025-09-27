package modular

import (
	"fmt"
	"strings"
)

// Decorator application steps for logger decorator BDD tests

func (ctx *LoggerDecoratorBDDTestContext) iApplyAPrefixDecoratorWithPrefix(prefix string) error {
	if ctx.currentLogger == nil {
		return errBaseLoggerNotSet
	}
	ctx.decoratedLogger = NewPrefixLoggerDecorator(ctx.currentLogger, prefix)
	ctx.currentLogger = ctx.decoratedLogger
	return nil
}

func (ctx *LoggerDecoratorBDDTestContext) iApplyAValueInjectionDecoratorWith(key1, value1 string) error {
	if ctx.currentLogger == nil {
		return errBaseLoggerNotSet
	}
	ctx.decoratedLogger = NewValueInjectionLoggerDecorator(ctx.currentLogger, key1, value1)
	ctx.currentLogger = ctx.decoratedLogger
	return nil
}

func (ctx *LoggerDecoratorBDDTestContext) iApplyAValueInjectionDecoratorWithTwoKeyValuePairs(key1, value1, key2, value2 string) error {
	if ctx.currentLogger == nil {
		return errBaseLoggerNotSet
	}
	ctx.decoratedLogger = NewValueInjectionLoggerDecorator(ctx.currentLogger, key1, value1, key2, value2)
	ctx.currentLogger = ctx.decoratedLogger
	return nil
}

func (ctx *LoggerDecoratorBDDTestContext) iApplyADualWriterDecorator() error {
	var primary, secondary Logger

	// Try different combinations of available loggers
	if ctx.primaryLogger != nil && ctx.secondaryLogger != nil {
		primary, secondary = ctx.primaryLogger, ctx.secondaryLogger
	} else if ctx.primaryLogger != nil && ctx.auditLogger != nil {
		primary, secondary = ctx.primaryLogger, ctx.auditLogger
	} else if ctx.baseLogger != nil && ctx.primaryLogger != nil {
		primary, secondary = ctx.baseLogger, ctx.primaryLogger
	} else if ctx.baseLogger != nil && ctx.auditLogger != nil {
		primary, secondary = ctx.baseLogger, ctx.auditLogger
	} else {
		return fmt.Errorf("dual writer decorator requires two loggers, but insufficient loggers are configured")
	}

	ctx.decoratedLogger = NewDualWriterLoggerDecorator(primary, secondary)
	ctx.currentLogger = ctx.decoratedLogger
	return nil
}

func (ctx *LoggerDecoratorBDDTestContext) iApplyAFilterDecoratorThatBlocksMessagesContaining(blockedText string) error {
	if ctx.currentLogger == nil {
		return errBaseLoggerNotSet
	}
	ctx.decoratedLogger = NewFilterLoggerDecorator(ctx.currentLogger, []string{blockedText}, nil, nil)
	ctx.currentLogger = ctx.decoratedLogger
	return nil
}

func (ctx *LoggerDecoratorBDDTestContext) iApplyAFilterDecoratorThatBlocksDebugLevelLogs() error {
	if ctx.currentLogger == nil {
		return errBaseLoggerNotSet
	}
	levelFilters := map[string]bool{"debug": false, "info": true, "warn": true, "error": true}
	ctx.decoratedLogger = NewFilterLoggerDecorator(ctx.currentLogger, nil, nil, levelFilters)
	ctx.currentLogger = ctx.decoratedLogger
	return nil
}

func (ctx *LoggerDecoratorBDDTestContext) iApplyAFilterDecoratorThatBlocksLogsWhereEquals(key, value string) error {
	if ctx.currentLogger == nil {
		return errBaseLoggerNotSet
	}
	keyFilters := map[string]string{key: value}
	ctx.decoratedLogger = NewFilterLoggerDecorator(ctx.currentLogger, nil, keyFilters, nil)
	ctx.currentLogger = ctx.decoratedLogger
	return nil
}

func (ctx *LoggerDecoratorBDDTestContext) iApplyAFilterDecoratorThatAllowsOnlyLevels(levels string) error {
	if ctx.currentLogger == nil {
		return errBaseLoggerNotSet
	}

	// Parse level names from Gherkin format like '"info" and "error"'
	// Extract quoted level names
	var levelList []string
	parts := strings.Split(levels, `"`)
	for i, part := range parts {
		// Every odd index (1, 3, 5...) contains the quoted content
		if i%2 == 1 && strings.TrimSpace(part) != "" {
			levelList = append(levelList, strings.TrimSpace(part))
		}
	}

	levelFilters := map[string]bool{
		"debug": false,
		"info":  false,
		"warn":  false,
		"error": false,
	}
	for _, level := range levelList {
		levelFilters[level] = true
	}
	ctx.decoratedLogger = NewFilterLoggerDecorator(ctx.currentLogger, nil, nil, levelFilters)
	ctx.currentLogger = ctx.decoratedLogger
	return nil
}

func (ctx *LoggerDecoratorBDDTestContext) iApplyALevelModifierDecoratorThatMapsTo(fromLevel, toLevel string) error {
	if ctx.currentLogger == nil {
		return errBaseLoggerNotSet
	}
	levelMappings := map[string]string{fromLevel: toLevel}
	ctx.decoratedLogger = NewLevelModifierLoggerDecorator(ctx.currentLogger, levelMappings)
	ctx.currentLogger = ctx.decoratedLogger
	return nil
}

// Logging action steps

func (ctx *LoggerDecoratorBDDTestContext) iLogAnInfoMessage(message string) error {
	if ctx.currentLogger == nil {
		return errLoggerNotSet
	}
	ctx.currentLogger.Info(message)
	return nil
}

func (ctx *LoggerDecoratorBDDTestContext) iLogAnInfoMessageWithArgs(message, key, value string) error {
	if ctx.currentLogger == nil {
		return errLoggerNotSet
	}
	ctx.currentLogger.Info(message, key, value)
	return nil
}

func (ctx *LoggerDecoratorBDDTestContext) iLogADebugMessage(message string) error {
	if ctx.currentLogger == nil {
		return errLoggerNotSet
	}
	ctx.currentLogger.Debug(message)
	return nil
}

func (ctx *LoggerDecoratorBDDTestContext) iLogAWarnMessage(message string) error {
	if ctx.currentLogger == nil {
		return errLoggerNotSet
	}
	ctx.currentLogger.Warn(message)
	return nil
}

func (ctx *LoggerDecoratorBDDTestContext) iLogAnErrorMessage(message string) error {
	if ctx.currentLogger == nil {
		return errLoggerNotSet
	}
	ctx.currentLogger.Error(message)
	return nil
}

// SetLogger scenario steps

func (ctx *LoggerDecoratorBDDTestContext) iCreateADecoratedLoggerWithPrefix(prefix string) error {
	if ctx.initialLogger == nil {
		return errBaseLoggerNotSet
	}
	ctx.decoratedLogger = NewPrefixLoggerDecorator(ctx.initialLogger, prefix)
	return nil
}

func (ctx *LoggerDecoratorBDDTestContext) iSetTheDecoratedLoggerOnTheApplication() error {
	if ctx.decoratedLogger == nil {
		return errDecoratedLoggerNotSet
	}
	ctx.app.SetLogger(ctx.decoratedLogger)
	return nil
}

func (ctx *LoggerDecoratorBDDTestContext) iGetTheLoggerServiceFromTheApplication() error {
	var serviceLogger Logger
	err := ctx.app.GetService("logger", &serviceLogger)
	if err != nil {
		return err
	}
	ctx.currentLogger = serviceLogger
	return nil
}
