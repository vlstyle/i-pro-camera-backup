package fetcher

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vlstyle/i-pro-camera-downloader/internal/camera"
	"github.com/vlstyle/i-pro-camera-downloader/internal/store"
)

type fakeCamera struct {
	recs      []camera.Recording
	failOn    string
	listFroms []time.Time
}

func (f *fakeCamera) List(ctx context.Context, from time.Time, max int) ([]camera.Recording, error) {
	f.listFroms = append(f.listFroms, from)
	var out []camera.Recording
	for _, r := range f.recs {
		if !r.End.Before(from) {
			out = append(out, r)
		}
		if len(out) == max {
			break
		}
	}
	return out, nil
}

func (f *fakeCamera) Download(ctx context.Context, name string) (io.ReadCloser, error) {
	if name == f.failOn {
		return nil, errors.New("download failed")
	}
	return io.NopCloser(strings.NewReader("video:" + name)), nil
}

func rec(seq int, end string) camera.Recording {
	e, err := camera.ParseTime(end)
	if err != nil {
		panic(err)
	}
	return camera.Recording{
		Name:   fmt.Sprintf("%d_%s_A.mp4", seq, e.Add(9*time.Hour).Format("060102150405")),
		Start:  e.Add(-15 * time.Second),
		End:    e,
		EndRaw: end,
	}
}

func newFetcher(cam Camera, st store.Store, now time.Time) *Fetcher {
	return &Fetcher{
		Camera:     cam,
		CameraName: "camera1",
		Store:      st,
		Now:        func() time.Time { return now },
		PageSize:   2,
		Log:        slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}

func TestRunSavesAllAndPages(t *testing.T) {
	root := t.TempDir()
	st := store.NewLocal(root, "camera1")
	cam := &fakeCamera{recs: []camera.Recording{
		rec(1, "20260911T000010.50Z"),
		rec(2, "20260911T000100.00Z"),
		rec(3, "20260911T000200.99Z"),
		rec(4, "20260911T000300.00Z"),
		rec(5, "20260911T000400.10Z"),
	}}
	now := time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC)
	if err := newFetcher(cam, st, now).Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	for _, r := range cam.recs {
		key, _ := Key("camera1", r)
		got, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(key)))
		if err != nil || string(got) != "video:"+r.Name {
			t.Errorf("%s: got=%q err=%v", key, got, err)
		}
	}
	c, _, _ := st.LoadCursor(context.Background())
	if c.LastEndTime != "20260911T000400.10Z" {
		t.Errorf("cursor = %q", c.LastEndTime)
	}
	if want := time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC); !cam.listFroms[0].Equal(want) {
		t.Errorf("first from = %v, want %v", cam.listFroms[0], want)
	}
	for i, want := range []time.Time{
		time.Date(2026, 9, 11, 0, 1, 1, 0, time.UTC),
		time.Date(2026, 9, 11, 0, 3, 1, 0, time.UTC),
	} {
		if !cam.listFroms[i+1].Equal(want) {
			t.Errorf("from[%d] = %v, want %v", i+1, cam.listFroms[i+1], want)
		}
	}
}

func TestRunResumesFromCursor(t *testing.T) {
	root := t.TempDir()
	st := store.NewLocal(root, "camera1")
	cam := &fakeCamera{recs: []camera.Recording{
		rec(1, "20260911T000010.50Z"),
		rec(2, "20260911T000100.00Z"),
		rec(3, "20260911T000200.99Z"),
	}, failOn: rec(3, "20260911T000200.99Z").Name}
	now := time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC)

	if err := newFetcher(cam, st, now).Run(context.Background()); err == nil {
		t.Fatal("expected failure")
	}
	c, _, _ := st.LoadCursor(context.Background())
	if c.LastEndTime != "20260911T000100.00Z" {
		t.Fatalf("cursor after failure = %q", c.LastEndTime)
	}

	cam.failOn = ""
	cam.listFroms = nil
	if err := newFetcher(cam, st, now).Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if want := time.Date(2026, 9, 11, 0, 1, 1, 0, time.UTC); !cam.listFroms[0].Equal(want) {
		t.Errorf("resume from = %v, want %v", cam.listFroms[0], want)
	}
	c, _, _ = st.LoadCursor(context.Background())
	if c.LastEndTime != "20260911T000200.99Z" {
		t.Errorf("cursor = %q", c.LastEndTime)
	}
}

func TestRunNoNewRecordingsKeepsCursor(t *testing.T) {
	st := store.NewLocal(t.TempDir(), "camera1")
	cam := &fakeCamera{recs: []camera.Recording{rec(1, "20260911T000010.50Z")}}
	now := time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC)
	f := newFetcher(cam, st, now)
	if err := f.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := f.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	c, _, _ := st.LoadCursor(context.Background())
	if c.LastEndTime != "20260911T000010.50Z" {
		t.Errorf("cursor = %q", c.LastEndTime)
	}
}

func TestRunMaxFiles(t *testing.T) {
	st := store.NewLocal(t.TempDir(), "camera1")
	cam := &fakeCamera{recs: []camera.Recording{
		rec(1, "20260911T000010.50Z"),
		rec(2, "20260911T000100.00Z"),
		rec(3, "20260911T000200.99Z"),
	}}
	f := newFetcher(cam, st, time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC))
	f.MaxFiles = 2
	if err := f.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	c, _, _ := st.LoadCursor(context.Background())
	if c.LastEndTime != "20260911T000100.00Z" {
		t.Errorf("cursor = %q", c.LastEndTime)
	}
}

func TestKey(t *testing.T) {
	r := rec(10625451, "20260911T080005.99Z")
	key, err := Key("camera1", r)
	if err != nil {
		t.Fatal(err)
	}
	if want := "camera1/2026/202609/20260911/10625451_260911170005_A.mp4"; key != want {
		t.Errorf("key = %q, want %q", key, want)
	}

	r.Name = "weird.mp4"
	r.Start = time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	if key, _ := Key("camera1", r); key != "camera1/2026/202601/20260102/weird.mp4" {
		t.Errorf("fallback key = %q", key)
	}

	for _, bad := range []string{"../x.mp4", "a/b.mp4", ".hidden", ""} {
		r.Name = bad
		if _, err := Key("camera1", r); err == nil {
			t.Errorf("Key should reject %q", bad)
		}
	}
}
