package config

import (
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/vlstyle/i-pro-camera-backup/internal/store"
)

var cameraNamePattern = regexp.MustCompile(`^[a-z0-9_-]+$`)

type Config struct {
	CameraName     string
	CameraHost     string
	CameraUser     string
	CameraPassword string
	Dest           store.Dest
	S3Endpoint     string
	MaxFiles       int
	HTTPTimeout    time.Duration
	LogLevel       slog.Level
	LogFormat      string
}

func Load(lookup func(string) (string, bool)) (Config, error) {
	get := func(key, def string) string {
		if v, ok := lookup(key); ok && v != "" {
			return v
		}
		return def
	}
	var errs []error
	required := func(key string) string {
		v := get(key, "")
		if v == "" {
			errs = append(errs, fmt.Errorf("%s is required", key))
		}
		return v
	}

	c := Config{
		CameraName:     required("CAMERA_NAME"),
		CameraHost:     required("CAMERA_HOST"),
		CameraUser:     required("CAMERA_USER"),
		CameraPassword: required("CAMERA_PASSWORD"),
		S3Endpoint:     get("DEST_S3_ENDPOINT", ""),
		LogFormat:      strings.ToLower(get("LOG_FORMAT", "json")),
	}
	destURL := required("DEST_URL")

	if c.CameraName != "" && !cameraNamePattern.MatchString(c.CameraName) {
		errs = append(errs, errors.New("CAMERA_NAME must match [a-z0-9_-]+"))
	}
	if destURL != "" {
		d, err := store.ParseDest(destURL)
		if err != nil {
			errs = append(errs, err)
		}
		c.Dest = d
	}

	n, err := strconv.Atoi(get("FETCH_MAX_FILES", "0"))
	if err != nil || n < 0 {
		errs = append(errs, errors.New("FETCH_MAX_FILES must be a non-negative integer"))
	}
	c.MaxFiles = n

	c.HTTPTimeout, err = time.ParseDuration(get("FETCH_HTTP_TIMEOUT", "60s"))
	if err != nil || c.HTTPTimeout <= 0 {
		errs = append(errs, errors.New("FETCH_HTTP_TIMEOUT must be a positive duration"))
	}

	if err := c.LogLevel.UnmarshalText([]byte(get("LOG_LEVEL", "info"))); err != nil {
		errs = append(errs, errors.New("LOG_LEVEL must be one of debug, info, warn, error"))
	}
	if c.LogFormat != "json" && c.LogFormat != "text" {
		errs = append(errs, errors.New("LOG_FORMAT must be json or text"))
	}

	if err := errors.Join(errs...); err != nil {
		return Config{}, err
	}
	return c, nil
}
