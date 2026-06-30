package v1

import (
	logV1 "github.com/ClessLi/component-base/pkg/log/v1"
	kitlog "github.com/go-kit/kit/log"
)

// Logger instances for different log levels.
var (
	infoLogger  kitlog.Logger // infoLogger handles normal operation logs
	errorLogger kitlog.Logger // errorLogger handles error-level logs
	panicLogger kitlog.Logger // panicLogger handles panic-level logs
	limit       int           // limit is the maximum length for truncated result messages
)

// getLimitResult truncates a byte slice to the configured limit and appends "...".
// Used for limiting the size of result messages in logs.
func getLimitResult(result []byte) string {
	var formattedRet []byte
	if len(result)-1 > limit+3 {
		formattedRet = result[:limit]
	} else {
		formattedRet = result
	}

	return string(formattedRet) + "..."
}

// init initializes logger instances and default configuration.
// This function runs once at package load time.
func init() {
	infoLogger = logV1.K()
	// Set errorLogger to ERROR level to ensure error logs are always recorded
	// Note: V(-2) means zapcore.ErrorLevel (4), because V() uses -1 * level
	errorLogger = logV1.V(-2).KitLogger()
	// Set panicLogger to PANIC level to ensure panic logs are always recorded
	// Note: V(-4) means zapcore.PanicLevel (4), because V() uses -1 * level
	panicLogger = logV1.V(-4).KitLogger()
	limit = 200
}
