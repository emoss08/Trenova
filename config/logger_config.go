package config

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/pkgerrors"
	"github.com/valyala/fasthttp"
	"gopkg.in/natefinch/lumberjack.v2"
)

type LoggerConfig struct {
	Level              int
	PrettyPrintConsole bool
	LogToFile          bool
	LogFilePath        string
	LogMaxSize         int
	LogMaxBackups      int
	LogMaxAge          int
	LogCompress        bool
}

// ServerLogger is a wrapper around zerolog.Logger that includes additional methods
type ServerLogger struct {
	*zerolog.Logger
}

func NewLogger(config LoggerConfig) *ServerLogger {
	// Set global logging level

	zerolog.SetGlobalLevel(zerolog.Level(config.Level))

	// Enable stack trace marshaling
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack

	// Create a multi-writer for console and file logging
	var writers []io.Writer
	// Console writer

	if config.PrettyPrintConsole {
		writers = append(writers, zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
			FormatLevel: func(i interface{}) string {
				return fmt.Sprintf("| %-6s|", i)
			},
		})
	} else {
		writers = append(writers, os.Stdout)
	}

	// File writer
	if config.LogToFile {
		writers = append(writers, &lumberjack.Logger{
			Filename:   config.LogFilePath,
			MaxSize:    config.LogMaxSize,    // megabytes
			MaxBackups: config.LogMaxBackups, // number of backups
			MaxAge:     config.LogMaxAge,     // days
			Compress:   config.LogCompress,   // compress rotated files
		})
	}

	// Create multi-writer
	mw := io.MultiWriter(writers...)

	// Create logger
	zl := zerolog.New(mw).With().Timestamp().Caller().Logger()

	return &ServerLogger{&zl}
}

func (l *ServerLogger) LogHTTP(r *fasthttp.Request, status, size int, duration time.Duration) {
	l.Info().
		Str("method", string(r.Header.Method())).
		Str("url", string(r.URI().FullURI())).
		Int("status", status).
		Int("size", size).
		Dur("duration", duration).
		Msg("HTTP request")
}

func (l *ServerLogger) LogAppInfo(version, environment string) {
	l.Info().
		Str("version", version).
		Str("environment", environment).
		Str("go_version", runtime.Version()).
		Str("os", runtime.GOOS).
		Str("arch", runtime.GOARCH).
		Str("num_cpu", strconv.Itoa(runtime.NumCPU())).
		Str("num_goroutine", strconv.Itoa(runtime.NumGoroutine())).
		Msg("Application started")
}

func (l *ServerLogger) LogError(err error, message string) {
	l.Error().
		Err(err).
		Str("stack_trace", fmt.Sprintf("%+v", err)).
		Msg(message)
}

func (l *ServerLogger) LogFatal(err error, message string) {
	l.Fatal().
		Err(err).
		Str("stack_trace", fmt.Sprintf("%+v", err)).
		Msg(message)
}
