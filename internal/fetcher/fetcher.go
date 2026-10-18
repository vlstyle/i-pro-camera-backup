package fetcher

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"regexp"
	"time"

	"github.com/vlstyle/i-pro-camera-downloader/internal/camera"
	"github.com/vlstyle/i-pro-camera-downloader/internal/store"
)

const DefaultPageSize = 100

var (
	fileNamePattern  = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]*$`)
	localDatePattern = regexp.MustCompile(`^\d+_(\d{2})(\d{2})(\d{2})\d{6}_[A-Za-z0-9]+\.mp4$`)
)

type Camera interface {
	List(ctx context.Context, from time.Time, max int) ([]camera.Recording, error)
	Download(ctx context.Context, name string) (io.ReadCloser, error)
}

type Fetcher struct {
	Camera     Camera
	CameraName string
	Store      store.Store
	Now        func() time.Time
	MaxFiles   int
	PageSize   int
	Log        *slog.Logger
}

func (f *Fetcher) Run(ctx context.Context) error {
	release, err := f.Store.Acquire(ctx)
	if err != nil {
		return err
	}
	defer release()

	cursor, version, err := f.Store.LoadCursor(ctx)
	var from time.Time
	switch {
	case errors.Is(err, store.ErrNoCursor):
		from = startOfPreviousDay(f.Now())
		f.Log.Info("no cursor found, starting from previous day", "from", from.Format(camera.TimeLayout))
	case err != nil:
		return fmt.Errorf("load cursor: %w", err)
	default:
		end, perr := camera.ParseTime(cursor.LastEndTime)
		if perr != nil {
			return fmt.Errorf("invalid last_end_time %q: %w", cursor.LastEndTime, perr)
		}
		from = nextSecond(end)
		f.Log.Info("cursor loaded", "last_end_time", cursor.LastEndTime)
	}

	pageSize := f.PageSize
	if pageSize <= 0 {
		pageSize = DefaultPageSize
	}

	saved := 0
	for {
		recs, err := f.Camera.List(ctx, from, pageSize)
		if err != nil {
			return fmt.Errorf("list recordings: %w", err)
		}
		f.Log.Debug("listed recordings", "from", from.Format(camera.TimeLayout), "count", len(recs))
		if len(recs) == 0 {
			break
		}
		for _, rec := range recs {
			if f.MaxFiles > 0 && saved >= f.MaxFiles {
				f.Log.Info("reached max files, stopping", "saved", saved)
				return nil
			}
			key, err := Key(f.CameraName, rec)
			if err != nil {
				return err
			}
			size, err := f.save(ctx, rec, key)
			if err != nil {
				return err
			}
			version, err = f.Store.SaveCursor(ctx, store.Cursor{LastEndTime: rec.EndRaw}, version)
			if err != nil {
				return fmt.Errorf("save cursor: %w", err)
			}
			saved++
			f.Log.Info("saved recording", "file", rec.Name, "key", key, "bytes", size, "end_time", rec.EndRaw)
		}
		if len(recs) < pageSize {
			break
		}
		from = nextSecond(recs[len(recs)-1].End)
	}
	f.Log.Info("finished", "saved", saved)
	return nil
}

type countingReader struct {
	r io.Reader
	n int64
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n += int64(n)
	return n, err
}

func (f *Fetcher) save(ctx context.Context, rec camera.Recording, key string) (int64, error) {
	body, err := f.Camera.Download(ctx, rec.Name)
	if err != nil {
		return 0, err
	}
	defer body.Close()
	cr := &countingReader{r: body}
	if err := f.Store.PutVideo(ctx, key, cr); err != nil {
		return 0, fmt.Errorf("store %s: %w", rec.Name, err)
	}
	return cr.n, nil
}

func Key(cameraName string, rec camera.Recording) (string, error) {
	if !fileNamePattern.MatchString(rec.Name) {
		return "", fmt.Errorf("unsafe file name %q", rec.Name)
	}
	year, month, day := "", "", ""
	if m := localDatePattern.FindStringSubmatch(rec.Name); m != nil {
		year, month, day = "20"+m[1], m[2], m[3]
	} else {
		t := rec.Start.UTC()
		year, month, day = t.Format("2006"), t.Format("01"), t.Format("02")
	}
	return fmt.Sprintf("%s/%s/%s%s/%s%s%s/%s", cameraName, year, year, month, year, month, day, rec.Name), nil
}

func nextSecond(t time.Time) time.Time {
	return t.UTC().Truncate(time.Second).Add(time.Second)
}

func startOfPreviousDay(now time.Time) time.Time {
	n := now.UTC()
	return time.Date(n.Year(), n.Month(), n.Day()-1, 0, 0, 0, 0, time.UTC)
}
