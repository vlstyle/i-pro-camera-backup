package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
)

type Local struct {
	root   string
	camera string
}

func NewLocal(root, camera string) *Local {
	return &Local{root: root, camera: camera}
}

func (l *Local) Acquire(ctx context.Context) (func(), error) {
	dir := filepath.Join(l.root, "_state")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(filepath.Join(dir, l.camera+".lock"), os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		f.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return nil, ErrLocked
		}
		return nil, err
	}
	return func() {
		syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		f.Close()
	}, nil
}

func (l *Local) PutVideo(ctx context.Context, key string, body io.Reader) error {
	return writeAtomic(filepath.Join(l.root, filepath.FromSlash(key)), body)
}

func (l *Local) LoadCursor(ctx context.Context) (Cursor, string, error) {
	data, err := os.ReadFile(filepath.Join(l.root, filepath.FromSlash(cursorKey(l.camera))))
	if errors.Is(err, fs.ErrNotExist) {
		return Cursor{}, "", ErrNoCursor
	}
	if err != nil {
		return Cursor{}, "", err
	}
	var c Cursor
	if err := json.Unmarshal(data, &c); err != nil {
		return Cursor{}, "", fmt.Errorf("invalid cursor file: %w", err)
	}
	return c, "", nil
}

func (l *Local) SaveCursor(ctx context.Context, c Cursor, _ string) (string, error) {
	data, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	data = append(data, '\n')
	return "", writeAtomic(filepath.Join(l.root, filepath.FromSlash(cursorKey(l.camera))), bytesReader(data))
}

func writeAtomic(dst string, body io.Reader) error {
	dir := filepath.Dir(dst)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := io.Copy(tmp, body); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmp.Name(), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), dst)
}
