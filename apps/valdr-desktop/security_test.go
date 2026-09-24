package main

import (
	"os"
	"strings"
	"testing"
)

func TestDesktopFrontendContentSecurityPolicy(t *testing.T) {
	raw, err := os.ReadFile("frontend/index.html")
	if err != nil {
		t.Fatal(err)
	}
	html := string(raw)

	required := []string{
		"Content-Security-Policy",
		"default-src 'self'",
		"base-uri 'none'",
		"object-src 'none'",
		"frame-src 'none'",
		"form-action 'none'",
		"img-src 'self' data:",
		"style-src 'self'",
		"script-src 'self'",
		"connect-src 'self'",
		"worker-src 'none'",
	}
	for _, token := range required {
		if !strings.Contains(html, token) {
			t.Fatalf("desktop CSP missing %q", token)
		}
	}

	forbidden := []string{
		"'unsafe-inline'",
		"'unsafe-eval'",
		"http://",
		"https://",
		"//cdn.",
	}
	for _, token := range forbidden {
		if strings.Contains(html, token) {
			t.Fatalf("desktop index contains forbidden remote/unsafe token %q", token)
		}
	}
}
