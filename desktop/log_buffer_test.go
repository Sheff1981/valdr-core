package desktop

import (
	"strings"
	"sync"
	"testing"
)

func TestDesktopLogBufferIsBoundedAndRedactsSensitiveLines(t *testing.T) {
	buffer := NewLogBuffer(96)

	if _, err := buffer.Write([]byte(
		"2026/09/25 [NODE] ready\npassword=never-show-this\n",
	)); err != nil {
		t.Fatal(err)
	}
	if _, err := buffer.Write([]byte(strings.Repeat("x", 120))); err != nil {
		t.Fatal(err)
	}

	got := buffer.String()
	if len([]byte(got)) > 96 {
		t.Fatalf("log buffer bytes=%d want <=96", len([]byte(got)))
	}
	if strings.Contains(got, "never-show-this") {
		t.Fatalf("log buffer leaked sensitive value: %q", got)
	}
}

func TestDesktopLogBufferConcurrentWrites(t *testing.T) {
	buffer := NewLogBuffer(4096)
	var wg sync.WaitGroup

	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				_, _ = buffer.Write([]byte("[NODE] concurrent log line\n"))
			}
		}()
	}
	wg.Wait()

	got := buffer.String()
	if got == "" {
		t.Fatal("concurrent log buffer unexpectedly empty")
	}
	if len([]byte(got)) > 4096 {
		t.Fatalf("log buffer bytes=%d want <=4096", len([]byte(got)))
	}
}

func TestDesktopLogBufferClear(t *testing.T) {
	buffer := NewLogBuffer(128)
	_, _ = buffer.Write([]byte("[NODE] test\n"))
	buffer.Clear()
	if got := buffer.String(); got != "" {
		t.Fatalf("Clear left log data: %q", got)
	}
}
