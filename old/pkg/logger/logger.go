package logger

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	initialized = false
	once        sync.Once
)

// Init initializes the logger with the given application name and log level.
func Init(logLevel string) {
	if initialized {
		return
	}

	if len(logLevel) == 0 {
		log.Warn().Msg("Log level is not set, defaulting to WARN")
		logLevel = "WARN"
	}

	once.Do(func() {
		zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
		zerolog.SetGlobalLevel(parseLogLevel(logLevel))

		log.Logger = log.With().
			Caller().
			Logger().Output(zerolog.ConsoleWriter{
			Out:           os.Stdout,
			TimeFormat:    "2006-01-02 15:04:05",
			NoColor:       true,
			FormatLevel:   func(i any) string { return strings.ToUpper(fmt.Sprintf("- [%s] -", i)) },
			FormatCaller:  func(i any) string { return fmt.Sprintf("%s", i) },
			FormatMessage: func(i any) string { return fmt.Sprintf("%s", i) },
			PartsOrder: []string{
				zerolog.TimestampFieldName,
				zerolog.LevelFieldName,
				zerolog.CallerFieldName,
				zerolog.MessageFieldName,
			},
		})

		zerolog.CallerMarshalFunc = func(pc uintptr, file string, line int) string {
			parts := strings.Split(file, "/")
			if len(parts) == 1 {
				return fmt.Sprintf("[%s::%d]", parts[0], line)
			}

			return fmt.Sprintf("[%s::%d]", parts[len(parts)-1], line)
		}

		initialized = true
		log.Info().Msg("Logger initialized successfully")
	})
}

// parseLogLevel converts a log level string to a zerolog.Level.
func parseLogLevel(level string) zerolog.Level {
	switch level {
	case "DEBUG", "debug":
		return zerolog.DebugLevel
	case "INFO", "info":
		return zerolog.InfoLevel
	case "WARN", "warn":
		return zerolog.WarnLevel
	case "ERROR", "error":
		return zerolog.ErrorLevel
	case "FATAL", "fatal":
		return zerolog.FatalLevel
	case "PANIC", "panic":
		return zerolog.PanicLevel
	default:
		log.Warn().Str("logLevel", level).Msg("Invalid log level, defaulting to WARN")
		return zerolog.WarnLevel
	}
}
