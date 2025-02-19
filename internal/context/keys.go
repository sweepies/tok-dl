package context

// Key is used for type-safe context values
type Key string

const (
	// LoggerKey is the context key for the application logger
	LoggerKey Key = "logger"
)