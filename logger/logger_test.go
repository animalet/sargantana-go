package logger

import (
	"bytes"
	"log"
	"strings"
	"testing"
)

func TestLogLevels(t *testing.T) {
	var buf bytes.Buffer
	SetOutput(&buf)
	defer SetOutput(log.Writer())

	tests := []struct {
		name      string
		level     LogLevel
		message   string
		shouldLog map[LogLevel]bool
	}{
		{
			name:    "debug level logs everything",
			level:   DEBUG,
			message: "test message",
			shouldLog: map[LogLevel]bool{
				DEBUG: true,
				INFO:  true,
				WARN:  true,
				ERROR: true,
				FATAL: true,
			},
		},
		{
			name:    "info level logs info and above",
			level:   INFO,
			message: "test message",
			shouldLog: map[LogLevel]bool{
				DEBUG: false,
				INFO:  true,
				WARN:  true,
				ERROR: true,
				FATAL: true,
			},
		},
		{
			name:    "warn level logs warn and above",
			level:   WARN,
			message: "test message",
			shouldLog: map[LogLevel]bool{
				DEBUG: false,
				INFO:  false,
				WARN:  true,
				ERROR: true,
				FATAL: true,
			},
		},
		{
			name:    "error level logs error and above",
			level:   ERROR,
			message: "test message",
			shouldLog: map[LogLevel]bool{
				DEBUG: false,
				INFO:  false,
				WARN:  false,
				ERROR: true,
				FATAL: true,
			},
		},
		{
			name:    "fatal level logs only fatal",
			level:   FATAL,
			message: "test message",
			shouldLog: map[LogLevel]bool{
				DEBUG: false,
				INFO:  false,
				WARN:  false,
				ERROR: false,
				FATAL: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetLevel(tt.level)

			// Test each log function
			buf.Reset()
			Debug(tt.message)
			output := buf.String()
			if tt.shouldLog[DEBUG] && !strings.Contains(output, "DEBUG") {
				t.Errorf("Expected DEBUG log to be written")
			}
			if !tt.shouldLog[DEBUG] && strings.Contains(output, "DEBUG") {
				t.Errorf("Expected DEBUG log to be filtered out")
			}

			buf.Reset()
			Info(tt.message)
			output = buf.String()
			if tt.shouldLog[INFO] && !strings.Contains(output, "INFO") {
				t.Errorf("Expected INFO log to be written")
			}
			if !tt.shouldLog[INFO] && strings.Contains(output, "INFO") {
				t.Errorf("Expected INFO log to be filtered out")
			}

			buf.Reset()
			Warn(tt.message)
			output = buf.String()
			if tt.shouldLog[WARN] && !strings.Contains(output, "WARN") {
				t.Errorf("Expected WARN log to be written")
			}
			if !tt.shouldLog[WARN] && strings.Contains(output, "WARN") {
				t.Errorf("Expected WARN log to be filtered out")
			}

			buf.Reset()
			Error(tt.message)
			output = buf.String()
			if tt.shouldLog[ERROR] && !strings.Contains(output, "ERROR") {
				t.Errorf("Expected ERROR log to be written")
			}
			if !tt.shouldLog[ERROR] && strings.Contains(output, "ERROR") {
				t.Errorf("Expected ERROR log to be filtered out")
			}

			// Test Fatal function
			buf.Reset()
			Fatal(tt.message)
			output = buf.String()
			if tt.shouldLog[FATAL] && !strings.Contains(output, "FATAL") {
				t.Errorf("Expected FATAL log to be written")
			}
			if !tt.shouldLog[FATAL] && strings.Contains(output, "FATAL") {
				t.Errorf("Expected FATAL log to be filtered out")
			}
		})
	}
}

func TestLogFormatting(t *testing.T) {
	var buf bytes.Buffer
	SetOutput(&buf)
	defer SetOutput(log.Writer())
	SetLevel(DEBUG)

	tests := []struct {
		name     string
		logFunc  func(string, ...interface{})
		format   string
		args     []interface{}
		expected string
		level    string
	}{
		{
			name:     "debug formatted message",
			logFunc:  Debugf,
			format:   "processing %d items in %s mode",
			args:     []interface{}{42, "test"},
			expected: "processing 42 items in test mode",
			level:    "DEBUG",
		},
		{
			name:     "info formatted message",
			logFunc:  Infof,
			format:   "user %s logged in with ID %d",
			args:     []interface{}{"john", 123},
			expected: "user john logged in with ID 123",
			level:    "INFO",
		},
		{
			name:     "warn formatted message",
			logFunc:  Warnf,
			format:   "warning: %s service is down, retrying in %d seconds",
			args:     []interface{}{"database", 30},
			expected: "warning: database service is down, retrying in 30 seconds",
			level:    "WARN",
		},
		{
			name:     "error formatted message",
			logFunc:  Errorf,
			format:   "failed to connect to %s:%d - %v",
			args:     []interface{}{"localhost", 5432, "connection refused"},
			expected: "failed to connect to localhost:5432 - connection refused",
			level:    "ERROR",
		},
		{
			name:     "fatal formatted message",
			logFunc:  Fatalf,
			format:   "fatal error: %s failed with code %d",
			args:     []interface{}{"startup", 1},
			expected: "fatal error: startup failed with code 1",
			level:    "FATAL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			tt.logFunc(tt.format, tt.args...)
			output := buf.String()

			if !strings.Contains(output, tt.expected) {
				t.Errorf("Expected output to contain %q, got %q", tt.expected, output)
			}

			if !strings.Contains(output, tt.level) {
				t.Errorf("Expected output to contain level %q, got %q", tt.level, output)
			}
		})
	}
}

func TestSetLevel(t *testing.T) {
	originalLevel := GetLevel()
	defer SetLevel(originalLevel)

	SetLevel(DEBUG)
	if GetLevel() != DEBUG {
		t.Errorf("Expected level DEBUG, got %v", GetLevel())
	}

	SetLevel(ERROR)
	if GetLevel() != ERROR {
		t.Errorf("Expected level ERROR, got %v", GetLevel())
	}
}

func TestGetLevel(t *testing.T) {
	originalLevel := GetLevel()
	defer SetLevel(originalLevel)

	SetLevel(WARN)
	level := GetLevel()
	if level != WARN {
		t.Errorf("Expected WARN, got %v", level)
	}
}

func TestLogLevelString(t *testing.T) {
	tests := []struct {
		level    LogLevel
		expected string
	}{
		{DEBUG, "DEBUG"},
		{INFO, "INFO"},
		{WARN, "WARN"},
		{ERROR, "ERROR"},
		{FATAL, "FATAL"},
		{LogLevel(999), "UNKNOWN"},
	}

	for _, test := range tests {
		result := test.level.String()
		if result != test.expected {
			t.Errorf("Expected %s, got %s", test.expected, result)
		}
	}
}

func TestSetFlags(t *testing.T) {
	var buf bytes.Buffer
	SetOutput(&buf)
	defer SetOutput(log.Writer())

	SetFlags(log.Lshortfile)
	SetLevel(DEBUG)
	Debug("test message")

	output := buf.String()
	if !strings.Contains(output, "test message") {
		t.Error("Expected log message to contain 'test message'")
	}
}

func TestSetOutput(t *testing.T) {
	var buf1, buf2 bytes.Buffer
	originalWriter := log.Writer()
	defer SetOutput(originalWriter)

	// Test setting output to first buffer
	SetOutput(&buf1)
	SetLevel(DEBUG)

	Debug("debug test")
	Info("info test")
	Warn("warn test")
	Error("error test")
	Fatal("fatal test")

	output := buf1.String()
	if !strings.Contains(output, "debug test") {
		t.Error("Expected output to contain 'debug test'")
	}
	if !strings.Contains(output, "info test") {
		t.Error("Expected output to contain 'info test'")
	}
	if !strings.Contains(output, "warn test") {
		t.Error("Expected output to contain 'warn test'")
	}
	if !strings.Contains(output, "error test") {
		t.Error("Expected output to contain 'error test'")
	}
	if !strings.Contains(output, "fatal test") {
		t.Error("Expected output to contain 'fatal test'")
	}

	// Test changing output to second buffer
	SetOutput(&buf2)
	Info("test message 2")

	if strings.Contains(buf1.String(), "test message 2") {
		t.Error("Expected second message NOT to be written to first buffer")
	}

	if !strings.Contains(buf2.String(), "test message 2") {
		t.Error("Expected second message to be written to second buffer")
	}
}

func TestFormattedLoggingLevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	originalWriter := log.Writer()
	originalLevel := GetLevel()

	// Ensure clean state
	SetOutput(&buf)
	defer SetOutput(originalWriter)
	defer SetLevel(originalLevel)

	// Set level to WARN - should log WARN, ERROR, and FATAL
	SetLevel(WARN)

	// Test DEBUG - should be filtered
	buf.Reset()
	Debugf("debug %s", "message")
	debugOutput := buf.String()
	if debugOutput != "" {
		t.Errorf("Debug message should be filtered at WARN level, got: %s", debugOutput)
	}

	// Test INFO - should be filtered
	buf.Reset()
	Infof("info %s", "message")
	infoOutput := buf.String()
	if infoOutput != "" {
		t.Errorf("Info message should be filtered at WARN level, got: %s", infoOutput)
	}

	// Test WARN - should appear
	buf.Reset()
	Warnf("warn %s", "message")
	warnOutput := buf.String()
	if warnOutput == "" || !strings.Contains(warnOutput, "warn message") {
		t.Errorf("Warn message should appear at WARN level, got: %s", warnOutput)
	}

	// Test ERROR - should appear
	buf.Reset()
	Errorf("error %s", "message")
	errorOutput := buf.String()
	if errorOutput == "" || !strings.Contains(errorOutput, "error message") {
		t.Errorf("Error message should appear at WARN level, got: %s", errorOutput)
	}

	// Test FATAL - should appear
	buf.Reset()
	Fatalf("fatal %s", "message")
	fatalOutput := buf.String()
	if fatalOutput == "" || !strings.Contains(fatalOutput, "fatal message") {
		t.Errorf("Fatal message should appear at WARN level, got: %s", fatalOutput)
	}
}

func TestLevelCombinations(t *testing.T) {
	var buf bytes.Buffer
	SetOutput(&buf)
	defer SetOutput(log.Writer())

	testCases := []struct {
		setLevel    LogLevel
		testLevel   LogLevel
		shouldLog   bool
		description string
	}{
		{DEBUG, DEBUG, true, "DEBUG level logs DEBUG messages"},
		{DEBUG, INFO, true, "DEBUG level logs INFO messages"},
		{DEBUG, WARN, true, "DEBUG level logs WARN messages"},
		{DEBUG, ERROR, true, "DEBUG level logs ERROR messages"},
		{INFO, DEBUG, false, "INFO level filters DEBUG messages"},
		{INFO, INFO, true, "INFO level logs INFO messages"},
		{INFO, WARN, true, "INFO level logs WARN messages"},
		{INFO, ERROR, true, "INFO level logs ERROR messages"},
		{WARN, DEBUG, false, "WARN level filters DEBUG messages"},
		{WARN, INFO, false, "WARN level filters INFO messages"},
		{WARN, WARN, true, "WARN level logs WARN messages"},
		{WARN, ERROR, true, "WARN level logs ERROR messages"},
		{ERROR, DEBUG, false, "ERROR level filters DEBUG messages"},
		{ERROR, INFO, false, "ERROR level filters INFO messages"},
		{ERROR, WARN, false, "ERROR level filters WARN messages"},
		{ERROR, ERROR, true, "ERROR level logs ERROR messages"},
		{FATAL, DEBUG, false, "FATAL level filters DEBUG messages"},
		{FATAL, INFO, false, "FATAL level filters INFO messages"},
		{FATAL, WARN, false, "FATAL level filters WARN messages"},
		{FATAL, ERROR, false, "FATAL level filters ERROR messages"},
		{FATAL, FATAL, true, "FATAL level logs FATAL messages"},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			SetLevel(tc.setLevel)
			buf.Reset()

			switch tc.testLevel {
			case DEBUG:
				Debugf("test message")
			case INFO:
				Infof("test message")
			case WARN:
				Warnf("test message")
			case ERROR:
				Errorf("test message")
			case FATAL:
				Fatalf("test message")
			}

			output := buf.String()
			if tc.shouldLog && output == "" {
				t.Errorf("Expected message to be logged but got empty output")
			}
			if !tc.shouldLog && output != "" {
				t.Errorf("Expected message to be filtered but got: %s", output)
			}
		})
	}
}
