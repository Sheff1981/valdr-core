package desktop

import (
	"strings"
	"sync"
	"unicode/utf8"
)

const DefaultDesktopLogBytes = 64 * 1024

var desktopSensitiveLogMarkers = []string{
	"passphrase",
	"password",
	"private_key",
	"private key",
	"auth_token",
	"auth token",
	"authorization",
}

// LogBuffer is a bounded, concurrency-safe writer used for local Desktop
// diagnostics. It never persists data and conservatively redacts any line
// that looks capable of carrying wallet or authorization secrets.
type LogBuffer struct {
	mu       sync.Mutex
	maxBytes int
	data     []byte
}

func NewLogBuffer(maxBytes int) *LogBuffer {
	if maxBytes <= 0 {
		maxBytes = DefaultDesktopLogBytes
	}
	return &LogBuffer{maxBytes: maxBytes}
}

func (b *LogBuffer) Write(p []byte) (int, error) {
	if b == nil {
		return len(p), nil
	}

	safe := []byte(redactDesktopLog(string(p)))

	b.mu.Lock()
	defer b.mu.Unlock()

	b.data = append(b.data, safe...)
	if len(b.data) <= b.maxBytes {
		return len(p), nil
	}

	start := len(b.data) - b.maxBytes
	for start < len(b.data) && !utf8.RuneStart(b.data[start]) {
		start++
	}
	trimmed := append([]byte(nil), b.data[start:]...)
	b.data = trimmed
	return len(p), nil
}

func (b *LogBuffer) String() string {
	if b == nil {
		return ""
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	return string(append([]byte(nil), b.data...))
}

func (b *LogBuffer) Clear() {
	if b == nil {
		return
	}
	b.mu.Lock()
	b.data = b.data[:0]
	b.mu.Unlock()
}

func redactDesktopLog(value string) string {
	parts := strings.SplitAfter(value, "\n")
	for i, part := range parts {
		lower := strings.ToLower(part)
		sensitive := false
		for _, marker := range desktopSensitiveLogMarkers {
			if strings.Contains(lower, marker) {
				sensitive = true
				break
			}
		}
		if !sensitive {
			continue
		}
		if strings.HasSuffix(part, "\n") {
			parts[i] = "[REDACTED sensitive log line]\n"
		} else {
			parts[i] = "[REDACTED sensitive log line]"
		}
	}
	return strings.Join(parts, "")
}
