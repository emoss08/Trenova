package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/pkg/config"
	"github.com/emoss08/trenova/internal/pkg/utils/fileutils"
	"github.com/logrusorgru/aurora"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/diode"
	oopszerolog "github.com/samber/oops/loggers/zerolog"
	"gopkg.in/natefinch/lumberjack.v2"
)

type Logger struct {
	*zerolog.Logger
	config *config.LogConfig
}

func NewLogger(cfg *config.Config) *Logger {
	logCfg := &config.LogConfig{
		Level:            cfg.Log.Level,
		SamplingPeriod:   cfg.Log.SamplingPeriod,
		SamplingInterval: cfg.Log.SamplingInterval,
		FileConfig: config.FileConfig{
			Enabled:    cfg.Log.FileConfig.Enabled,
			FileName:   cfg.Log.FileConfig.FileName,
			Path:       cfg.Log.FileConfig.Path,
			MaxSize:    cfg.Log.FileConfig.MaxSize,
			MaxBackups: cfg.Log.FileConfig.MaxBackups,
			MaxAge:     cfg.Log.FileConfig.MaxAge,
			Compress:   cfg.Log.FileConfig.Compress,
		},
	}

	writers := []io.Writer{}

	consoleWriter := configureConsoleWriter(cfg.App.Environment)
	writers = append(writers, consoleWriter)

	if logCfg.FileConfig.Enabled {
		fileWriter := configureFileWriter(logCfg.FileConfig)
		writers = append(writers, fileWriter)
	}

	multiWriter := zerolog.MultiLevelWriter(writers...)

	asyncWriter := diode.NewWriter(multiWriter, 1000, 10*time.Millisecond, func(missed int) {
		fmt.Printf("Logger Dropped %d messages", missed)
	})

	level, err := zerolog.ParseLevel(logCfg.Level)
	if err != nil {
		level = zerolog.InfoLevel
	}

	zerolog.SetGlobalLevel(level)
	zerolog.TimeFieldFormat = time.RFC3339Nano
	zerolog.CallerMarshalFunc = customCallerFormatter
	zerolog.ErrorStackMarshaler = oopszerolog.OopsStackMarshaller
	zerolog.ErrorMarshalFunc = oopszerolog.OopsMarshalFunc

	baseLogger := zerolog.New(asyncWriter).
		Hook(newContextHook()).
		With().
		Caller().
		Timestamp().
		Str("app", cfg.App.Name).
		Str("version", cfg.App.Version).
		Str("hostname", getHostname()).
		Logger()

	// Configure sampling
	sampledLogger := baseLogger.Sample(&zerolog.LevelSampler{
		DebugSampler: &zerolog.BurstSampler{
			Burst:  5,               // Allow 5 debug messages
			Period: 1 * time.Second, // Per second
			NextSampler: &zerolog.BasicSampler{
				N: 100, // Then sample every 100th message
			},
		},
		InfoSampler: &zerolog.BurstSampler{
			Burst:  10,              // Allow 10 info messages
			Period: 1 * time.Second, // Per second
			NextSampler: &zerolog.BasicSampler{
				N: 50, // Then sample every 50th message
			},
		},
		WarnSampler: &zerolog.BurstSampler{
			Burst:  5,               // Allow 5 warning messages
			Period: 1 * time.Second, // Per second
			NextSampler: &zerolog.BasicSampler{
				N: 10, // Then sample every 10th message
			},
		},
		// Error and Fatal levels are not sampled - we want to catch all of these
	})

	return &Logger{
		Logger: &sampledLogger,
		config: logCfg,
	}
}

func configureConsoleWriter(env string) zerolog.ConsoleWriter {
	output := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.RFC3339,
		NoColor:    env != "development",
	}

	output.FormatLevel = func(i any) string {
		var l string
		if ll, ok := i.(string); ok {
			level := strings.ToUpper(ll)
			switch ll {
			case "trace":
				l = aurora.Magenta(level).String()
			case "debug":
				l = aurora.Blue(level).String()
			case "info":
				l = aurora.Green(level).String()
			case "warn":
				l = aurora.Yellow(level).String()
			case "error":
				l = aurora.Red(level).String()
			case "fatal":
				l = aurora.Red(level).Bold().String()
			case "panic":
				l = aurora.Red(level).Bold().String()
			default:
				l = level
			}
		}
		return fmt.Sprintf("| %-6s |", l)
	}

	output.FormatMessage = func(i any) string {
		return fmt.Sprintf("message=%s", i)
	}

	return output
}

func configureFileWriter(cfg config.FileConfig) io.Writer {
	return &lumberjack.Logger{
		Filename:   filepath.Join(cfg.Path, cfg.FileName),
		MaxSize:    cfg.MaxSize,
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAge,
		Compress:   cfg.Compress,
	}
}

func customCallerFormatter(pc uintptr, file string, line int) string {
	// Get the function name
	funcName := runtime.FuncForPC(pc).Name()
	funcName = filepath.Base(funcName)

	// Find project root from the current file's location
	projectRoot, err := fileutils.FindProjectRoot(file)
	if err != nil {
		// Fallback to just filename if we can't find project root
		return fmt.Sprintf("%s:%d %s()", filepath.Base(file), line, funcName)
	}

	// Get relative path from project root
	relPath, err := filepath.Rel(projectRoot, file)
	if err != nil {
		// Fallback to just filename if we can't get relative path
		return fmt.Sprintf("%s:%d %s()", filepath.Base(file), line, funcName)
	}

	return fmt.Sprintf("%s:%d %s()", relPath, line, funcName)
}

type contextHook struct{}

func newContextHook() contextHook {
	return contextHook{}
}

func (h contextHook) Run(e *zerolog.Event, level zerolog.Level, _ string) {
	if level <= zerolog.WarnLevel {
		e.Str("go_version", runtime.Version()).
			Int("go_routines", runtime.NumGoroutine()).
			Int("cpu", runtime.NumCPU())
	}
}

func getHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown hostname"
	}

	return hostname
}
