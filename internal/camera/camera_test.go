package camera

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func newTestClient(t *testing.T, h http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	c := New(strings.TrimPrefix(srv.URL, "http://"), "user", "pass", 5*time.Second)
	return c
}

func TestListParsesRecordings(t *testing.T) {
	var gotQuery string
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		u, p, ok := r.BasicAuth()
		if !ok || u != "user" || p != "pass" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		gotQuery = r.URL.RawQuery
		io.WriteString(w, "10625452_260911170034_A.mp4,20260911T080031.20Z,20260911T080046.00Z\r\n"+
			"10625451_260911165955_A.mp4,20260911T075952.20Z,20260911T080005.99Z\r\n"+
			"Mp4fileNum=2\r\n")
	})
	from := time.Date(2026, 9, 11, 8, 0, 6, 0, time.UTC)
	recs, err := c.List(context.Background(), from, 100)
	if err != nil {
		t.Fatal(err)
	}
	if gotQuery != "Datafrom=20260911T080006Z&Maxmatches=100" {
		t.Errorf("query = %q", gotQuery)
	}
	if len(recs) != 2 {
		t.Fatalf("len = %d", len(recs))
	}
	if recs[0].Name != "10625451_260911165955_A.mp4" || recs[0].EndRaw != "20260911T080005.99Z" {
		t.Errorf("recs[0] = %+v", recs[0])
	}
	if want := time.Date(2026, 9, 11, 8, 0, 5, 990_000_000, time.UTC); !recs[0].End.Equal(want) {
		t.Errorf("end = %v, want %v", recs[0].End, want)
	}
}

func TestListEmpty(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "MP4fileNum=0\n")
	})
	recs, err := c.List(context.Background(), time.Now(), 100)
	if err != nil || len(recs) != 0 {
		t.Fatalf("recs=%v err=%v", recs, err)
	}
}

func TestListErrors(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "ErrorNo=3\r\n")
	})
	if _, err := c.List(context.Background(), time.Now(), 1); !errors.Is(err, ErrBusy) {
		t.Errorf("err = %v, want ErrBusy", err)
	}

	c = newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
	if _, err := c.List(context.Background(), time.Now(), 1); err == nil {
		t.Error("expected error for 401")
	}

	c = newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "garbage\n")
	})
	if _, err := c.List(context.Background(), time.Now(), 1); err == nil {
		t.Error("expected error for malformed line")
	}
}

func TestDownload(t *testing.T) {
	payload := "\x00\x00\x00\x18ftypmp42" + strings.Repeat("x", 1000)
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("FileName") == "missing.mp4" {
			io.WriteString(w, "ErrorNo=4\r\n")
			return
		}
		io.WriteString(w, payload)
	})
	body, err := c.Download(context.Background(), "a.mp4")
	if err != nil {
		t.Fatal(err)
	}
	got, _ := io.ReadAll(body)
	body.Close()
	if string(got) != payload {
		t.Errorf("payload mismatch: %d bytes", len(got))
	}

	if _, err := c.Download(context.Background(), "missing.mp4"); err == nil || !strings.Contains(err.Error(), "ErrorNo=4") {
		t.Errorf("err = %v", err)
	}
}
