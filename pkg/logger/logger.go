package logger

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	// Logger is the global logger instance
	Logger zerolog.Logger
)

// Init initializes the logger with the given options
func Init(opts ...Option) {
	// Default options
	options := &Options{
		Level:     zerolog.InfoLevel,
		Pretty:    true,
		TimeField: "time",
		Output:    os.Stdout,
	}

	// Apply options
	for _, opt := range opts {
		opt(options)
	}

	// Configure time field
	zerolog.TimeFieldFormat = time.RFC3339
	zerolog.TimestampFieldName = options.TimeField

	// Configure output
	var output io.Writer = options.Output
	if options.Pretty {
		output = zerolog.ConsoleWriter{
			Out:        options.Output,
			TimeFormat: "15:04:05",
			NoColor:    !options.Colors,
		}
	}

	// Create logger
	Logger = zerolog.New(output).With().Timestamp().Logger().Level(options.Level)

	// Set global logger
	log.Logger = Logger
}

// Options contains logger configuration options
type Options struct {
	Level     zerolog.Level
	Pretty    bool
	Colors    bool
	TimeField string
	Output    io.Writer
}

// Option is a function that configures Options
type Option func(*Options)

// WithLevel sets the log level
func WithLevel(level zerolog.Level) Option {
	return func(o *Options) {
		o.Level = level
	}
}

// WithPretty enables or disables pretty logging
func WithPretty(pretty bool) Option {
	return func(o *Options) {
		o.Pretty = pretty
	}
}

// WithColors enables or disables colors in pretty logging
func WithColors(colors bool) Option {
	return func(o *Options) {
		o.Colors = colors
	}
}

// WithTimeField sets the time field name
func WithTimeField(field string) Option {
	return func(o *Options) {
		o.TimeField = field
	}
}

// WithOutput sets the output writer
func WithOutput(output io.Writer) Option {
	return func(o *Options) {
		o.Output = output
	}
}

// Debug logs a debug message
func Debug(msg string) {
	Logger.Debug().Msg(msg)
}

// Debugf logs a formatted debug message
func Debugf(format string, v ...interface{}) {
	Logger.Debug().Msg(fmt.Sprintf(format, v...))
}

// Info logs an info message
func Info(msg string) {
	Logger.Info().Msg(msg)
}

// Infof logs a formatted info message
func Infof(format string, v ...interface{}) {
	Logger.Info().Msg(fmt.Sprintf(format, v...))
}

// Warn logs a warning message
func Warn(msg string) {
	Logger.Warn().Msg(msg)
}

// Warnf logs a formatted warning message
func Warnf(format string, v ...interface{}) {
	Logger.Warn().Msg(fmt.Sprintf(format, v...))
}

// Error logs an error message
func Error(err error) {
	if err != nil {
		Logger.Error().Err(err).Msg(err.Error())
	}
}

// Errorf logs a formatted error message
func Errorf(format string, v ...interface{}) {
	Logger.Error().Msg(fmt.Sprintf(format, v...))
}

// Fatal logs a fatal message and exits
func Fatal(err error) {
	if err != nil {
		Logger.Fatal().Err(err).Msg(err.Error())
	}
}

// Fatalf logs a formatted fatal message and exits
func Fatalf(format string, v ...interface{}) {
	Logger.Fatal().Msg(fmt.Sprintf(format, v...))
}

// Event creates a colored event log for file system events
func Event(eventType, path string) {
	var event *zerolog.Event

	switch eventType {
	case "CREATE":
		event = Logger.Info().Str("type", eventType).Str("path", path)
	case "WRITE":
		event = Logger.Info().Str("type", eventType).Str("path", path)
	case "REMOVE":
		event = Logger.Warn().Str("type", eventType).Str("path", path)
	case "RENAME":
		event = Logger.Info().Str("type", eventType).Str("path", path)
	case "CHMOD":
		event = Logger.Debug().Str("type", eventType).Str("path", path)
	default:
		event = Logger.Info().Str("type", eventType).Str("path", path)
	}

	event.Msg("File event")
}
