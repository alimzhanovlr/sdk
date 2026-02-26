package httpclient

import (
	"fmt"
	"log"
	"os"
	"time"
)

// SimpleLogger простая реализация Logger
type SimpleLogger struct {
	logger *log.Logger
	level  LogLevel
}

type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	ERROR
)

// NewSimpleLogger создает простой логгер
func NewSimpleLogger(level LogLevel) *SimpleLogger {
	return &SimpleLogger{
		logger: log.New(os.Stdout, "", 0),
		level:  level,
	}
}

func (l *SimpleLogger) Debug(msg string, fields ...interface{}) {
	if l.level <= DEBUG {
		l.log("DEBUG", msg, fields...)
	}
}

func (l *SimpleLogger) Info(msg string, fields ...interface{}) {
	if l.level <= INFO {
		l.log("INFO", msg, fields...)
	}
}

func (l *SimpleLogger) Error(msg string, fields ...interface{}) {
	if l.level <= ERROR {
		l.log("ERROR", msg, fields...)
	}
}

func (l *SimpleLogger) log(level, msg string, fields ...interface{}) {
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	output := fmt.Sprintf("[%s] %s: %s", timestamp, level, msg)

	if len(fields) > 0 {
		output += " |"
		for i := 0; i < len(fields); i += 2 {
			if i+1 < len(fields) {
				output += fmt.Sprintf(" %v=%v", fields[i], fields[i+1])
			}
		}
	}

	l.logger.Println(output)
}
