package logging

import (
	"bytes"
	"strings"
	"testing"

	"ticketdoctor/config"
)

func TestNewFormatsAndLevels(t *testing.T) {
	var buf bytes.Buffer
	log, err := New(config.LoggingConfig{Level: "warn", Format: "json"}, &buf)
	if err != nil {
		t.Fatal(err)
	}
	log.Info("hidden")
	log.Warn("shown", "k", "v")
	out := buf.String()
	if strings.Contains(out, "hidden") || !strings.Contains(out, `"msg":"shown"`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestNewRejectsBadSettings(t *testing.T) {
	if _, err := New(config.LoggingConfig{Level: "loud", Format: "text"}, &bytes.Buffer{}); err == nil {
		t.Fatal("expected error for bad level")
	}
	if _, err := New(config.LoggingConfig{Level: "info", Format: "xml"}, &bytes.Buffer{}); err == nil {
		t.Fatal("expected error for bad format")
	}
}
