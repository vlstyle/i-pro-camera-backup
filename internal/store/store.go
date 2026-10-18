package store

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"path"
	"strings"
)

var (
	ErrNoCursor       = errors.New("cursor not found")
	ErrCursorConflict = errors.New("cursor was updated by another process")
	ErrLocked         = errors.New("another process is running for this camera")
)

type Cursor struct {
	LastEndTime string `json:"last_end_time"`
}

type Store interface {
	Acquire(ctx context.Context) (release func(), err error)
	PutVideo(ctx context.Context, key string, body io.Reader) error
	LoadCursor(ctx context.Context) (cursor Cursor, version string, err error)
	SaveCursor(ctx context.Context, cursor Cursor, version string) (newVersion string, err error)
}

type Dest struct {
	Scheme string
	Path   string
	Bucket string
	Prefix string
}

func ParseDest(raw string) (Dest, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return Dest{}, fmt.Errorf("invalid DEST_URL: %w", err)
	}
	switch u.Scheme {
	case "file":
		if u.Host != "" || u.Path == "" {
			return Dest{}, errors.New("DEST_URL must be file:///absolute/path")
		}
		return Dest{Scheme: "file", Path: path.Clean(u.Path)}, nil
	case "s3":
		if u.Host == "" {
			return Dest{}, errors.New("DEST_URL must be s3://bucket/prefix/")
		}
		prefix := strings.TrimPrefix(u.Path, "/")
		if prefix != "" && !strings.HasSuffix(prefix, "/") {
			prefix += "/"
		}
		return Dest{Scheme: "s3", Bucket: u.Host, Prefix: prefix}, nil
	default:
		return Dest{}, fmt.Errorf("unsupported DEST_URL scheme %q", u.Scheme)
	}
}

func Open(ctx context.Context, d Dest, s3Endpoint, camera string) (Store, error) {
	switch d.Scheme {
	case "file":
		return NewLocal(d.Path, camera), nil
	case "s3":
		return NewS3(ctx, d.Bucket, d.Prefix, s3Endpoint, camera)
	}
	return nil, fmt.Errorf("unsupported destination %q", d.Scheme)
}

func cursorKey(camera string) string {
	return "_state/" + camera + ".json"
}
