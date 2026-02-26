package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger wraps zap logger
type Logger struct {
	*zap.Logger
}

// Config for logger
type Config struct {
	Level      string
	Format     string
	OutputPath string
}

// New creates a new logger instance
func New(cfg Config) (*Logger, error) {
	// Parse level
	level := zapcore.InfoLevel
	if err := level.UnmarshalText([]byte(cfg.Level)); err != nil {
		level = zapcore.InfoLevel
	}

	// Encoder config
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// Choose encoder
	var encoder zapcore.Encoder
	if cfg.Format == "console" {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}

	// Output
	var output zapcore.WriteSyncer
	if cfg.OutputPath == "stdout" || cfg.OutputPath == "" {
		output = zapcore.AddSync(os.Stdout)
	} else {
		file, err := os.OpenFile(cfg.OutputPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, err
		}
		output = zapcore.AddSync(file)
	}

	core := zapcore.NewCore(encoder, output, level)
	zapLogger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))

	return &Logger{Logger: zapLogger}, nil
}

// WithFields adds fields to logger
func (l *Logger) WithFields(fields ...zap.Field) *Logger {
	return &Logger{Logger: l.With(fields...)}
}

// WithError adds error field
func (l *Logger) WithError(err error) *Logger {
	return &Logger{Logger: l.With(zap.Error(err))}
}

// WithTraceID adds trace ID field
func (l *Logger) WithTraceID(traceID string) *Logger {
	return &Logger{Logger: l.With(zap.String("trace_id", traceID))}
}

// WithRequestID adds request ID field
func (l *Logger) WithRequestID(requestID string) *Logger {
	return &Logger{Logger: l.With(zap.String("request_id", requestID))}
}

// Helper functions for zap fields
func String(key, val string) zap.Field {
	return zap.String(key, val)
}

func Int(key string, val int) zap.Field {
	return zap.Int(key, val)
}

func Error(err error) zap.Field {
	return zap.Error(err)
}

func Any(key string, val interface{}) zap.Field {
	return zap.Any(key, val)
}
