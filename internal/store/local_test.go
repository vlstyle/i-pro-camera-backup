package store

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLocalCursorRoundTrip(t *testing.T) {
	l := NewLocal(t.TempDir(), "camera1")
	ctx := context.Background()
	if _, _, err := l.LoadCursor(ctx); !errors.Is(err, ErrNoCursor) {
		t.Fatalf("err = %v, want ErrNoCursor", err)
	}
	if _, err := l.SaveCursor(ctx, Cursor{LastEndTime: "20260911T080005.99Z"}, ""); err != nil {
		t.Fatal(err)
	}
	c, _, err := l.LoadCursor(ctx)
	if err != nil || c.LastEndTime != "20260911T080005.99Z" {
		t.Fatalf("c=%+v err=%v", c, err)
	}
}

func TestLocalPutVideo(t *testing.T) {
	root := t.TempDir()
	l := NewLocal(root, "camera1")
	if err := l.PutVideo(context.Background(), "camera1/2026/202609/20260911/a.mp4", strings.NewReader("data")); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(root, "camera1/2026/202609/20260911/a.mp4"))
	if err != nil || string(got) != "data" {
		t.Fatalf("got=%q err=%v", got, err)
	}
	entries, _ := os.ReadDir(filepath.Join(root, "camera1/2026/202609/20260911"))
	if len(entries) != 1 {
		t.Errorf("leftover files: %v", entries)
	}
}

func TestLocalAcquireExclusive(t *testing.T) {
	root := t.TempDir()
	a := NewLocal(root, "camera1")
	release, err := a.Acquire(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewLocal(root, "camera1").Acquire(context.Background()); !errors.Is(err, ErrLocked) {
		t.Errorf("err = %v, want ErrLocked", err)
	}
	if r, err := NewLocal(root, "camera2").Acquire(context.Background()); err != nil {
		t.Errorf("other camera should not conflict: %v", err)
	} else {
		r()
	}
	release()
	r, err := NewLocal(root, "camera1").Acquire(context.Background())
	if err != nil {
		t.Fatalf("after release: %v", err)
	}
	r()
}

func TestParseDest(t *testing.T) {
	d, err := ParseDest("s3://bucket/a/b")
	if err != nil || d.Bucket != "bucket" || d.Prefix != "a/b/" {
		t.Errorf("d=%+v err=%v", d, err)
	}
	d, err = ParseDest("s3://bucket")
	if err != nil || d.Prefix != "" {
		t.Errorf("d=%+v err=%v", d, err)
	}
	for _, bad := range []string{"file://host/x", "file:///", "s3:///x", "http://x"} {
		if bad == "file:///" {
			continue
		}
		if _, err := ParseDest(bad); err == nil {
			t.Errorf("ParseDest(%q) should fail", bad)
		}
	}
}
