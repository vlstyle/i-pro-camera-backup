package config

import (
	"log/slog"
	"strings"
	"testing"
	"time"
)

func lookup(m map[string]string) func(string) (string, bool) {
	return func(k string) (string, bool) {
		v, ok := m[k]
		return v, ok
	}
}

func base() map[string]string {
	return map[string]string{
		"CAMERA_NAME":     "camera1",
		"CAMERA_HOST":     "camera1.example.invalid",
		"CAMERA_USER":     "changeme",
		"CAMERA_PASSWORD": "changeme",
		"DEST_URL":        "file:///data",
	}
}

func TestLoadDefaults(t *testing.T) {
	c, err := Load(lookup(base()))
	if err != nil {
		t.Fatal(err)
	}
	if c.MaxFiles != 0 || c.HTTPTimeout != 60*time.Second || c.LogLevel != slog.LevelInfo || c.LogFormat != "json" {
		t.Errorf("unexpected defaults: %+v", c)
	}
	if c.Dest.Scheme != "file" || c.Dest.Path != "/data" {
		t.Errorf("dest = %+v", c.Dest)
	}
}

func TestLoadS3(t *testing.T) {
	m := base()
	m["DEST_URL"] = "s3://bucket/camera"
	m["FETCH_MAX_FILES"] = "10"
	m["LOG_LEVEL"] = "debug"
	m["LOG_FORMAT"] = "TEXT"
	c, err := Load(lookup(m))
	if err != nil {
		t.Fatal(err)
	}
	if c.Dest.Bucket != "bucket" || c.Dest.Prefix != "camera/" || c.MaxFiles != 10 || c.LogLevel != slog.LevelDebug || c.LogFormat != "text" {
		t.Errorf("unexpected: %+v", c)
	}
}

func TestLoadErrors(t *testing.T) {
	cases := map[string]func(map[string]string){
		"CAMERA_NAME is required":       func(m map[string]string) { delete(m, "CAMERA_NAME") },
		"CAMERA_PASSWORD is required":   func(m map[string]string) { delete(m, "CAMERA_PASSWORD") },
		"DEST_URL is required":          func(m map[string]string) { delete(m, "DEST_URL") },
		"CAMERA_NAME must match":        func(m map[string]string) { m["CAMERA_NAME"] = "Camera 1" },
		"unsupported DEST_URL scheme":   func(m map[string]string) { m["DEST_URL"] = "ftp://x/y" },
		"FETCH_MAX_FILES must be":       func(m map[string]string) { m["FETCH_MAX_FILES"] = "-1" },
		"FETCH_HTTP_TIMEOUT must be":    func(m map[string]string) { m["FETCH_HTTP_TIMEOUT"] = "abc" },
		"LOG_LEVEL must be":             func(m map[string]string) { m["LOG_LEVEL"] = "verbose" },
		"LOG_FORMAT must be json or te": func(m map[string]string) { m["LOG_FORMAT"] = "xml" },
	}
	for want, mutate := range cases {
		m := base()
		mutate(m)
		_, err := Load(lookup(m))
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("want error containing %q, got %v", want, err)
		}
	}
}
