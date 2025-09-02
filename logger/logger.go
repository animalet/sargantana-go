// Package logger provides a simple level-based logging solution for the Sargantana Go framework.
// It wraps the standard library's log package to provide Debug, Info, Warn, and Error levels
// while maintaining simple printf-style formatting without structured logging.
package logger

import (
	"fmt"
	"io"
	"log"
	"os"
)

// LogLevel represents the logging level
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
	FATAL
)

var (
	currentLevel = INFO
	debugLogger  = log.New(os.Stdout, "DEBUG: ", log.Ldate|log.Ltime|log.Lmsgprefix)
	infoLogger   = log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime|log.Lmsgprefix)
	warnLogger   = log.New(os.Stdout, "WARN: ", log.Ldate|log.Ltime|log.Lmsgprefix)
	errorLogger  = log.New(os.Stderr, "ERROR: ", log.Ldate|log.Ltime|log.Lmsgprefix)
	fatalLogger  = log.New(os.Stderr, "FATAL: ", log.Ldate|log.Ltime|log.Lmsgprefix)
)

const calldepth = 3

// SetLevel sets the minimum log level that will be output
func SetLevel(level LogLevel) {
	currentLevel = level
}

// SetFlags sets the logging flags for all loggers
func SetFlags(f int) {
	debugLogger.SetFlags(f)
	infoLogger.SetFlags(f)
	warnLogger.SetFlags(f)
	errorLogger.SetFlags(f)
	fatalLogger.SetFlags(f)
}

// SetOutput sets the output destination for all loggers
func SetOutput(w io.Writer) {
	debugLogger.SetOutput(w)
	infoLogger.SetOutput(w)
	warnLogger.SetOutput(w)
	errorLogger.SetOutput(w)
	fatalLogger.SetOutput(w)
}

func Debug(msg string) {
	if currentLevel <= DEBUG {
		_ = debugLogger.Output(calldepth, msg)
	}
}

func Info(msg string) {
	if currentLevel <= INFO {
		_ = infoLogger.Output(calldepth, msg)
	}
}

func Warn(msg string) {
	if currentLevel <= WARN {
		_ = warnLogger.Output(calldepth, msg)
	}
}

func Error(msg string) {
	if currentLevel <= ERROR {
		_ = errorLogger.Output(calldepth, msg)
	}
}

func Fatal(msg string) {
	if currentLevel <= FATAL {
		_ = fatalLogger.Output(calldepth, msg)
	}
}

func Debugf(format string, v ...interface{}) {
	if currentLevel <= DEBUG {
		_ = debugLogger.Output(calldepth, fmt.Sprintf(format, v...))
	}
}

func Infof(format string, v ...interface{}) {
	if currentLevel <= INFO {
		_ = infoLogger.Output(calldepth, fmt.Sprintf(format, v...))
	}
}

func Warnf(format string, v ...interface{}) {
	if currentLevel <= WARN {
		_ = warnLogger.Output(calldepth, fmt.Sprintf(format, v...))
	}
}

func Errorf(format string, v ...interface{}) {
	if currentLevel <= ERROR {
		_ = errorLogger.Output(calldepth, fmt.Sprintf(format, v...))
	}
}

func Fatalf(format string, v ...interface{}) {
	if currentLevel <= FATAL {
		_ = fatalLogger.Output(calldepth, fmt.Sprintf(format, v...))
	}
}

// GetLevel returns the current logging level
func GetLevel() LogLevel {
	return currentLevel
}

// LevelString returns the string representation of a log level
func (l LogLevel) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	case FATAL:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}
