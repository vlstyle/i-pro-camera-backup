package camera

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

const TimeLayout = "20060102T150405Z"

var ErrBusy = errors.New("camera is busy")

type Recording struct {
	Name   string
	Start  time.Time
	End    time.Time
	EndRaw string
}

type Client struct {
	base string
	user string
	pass string
	http *http.Client
}

func New(host, user, password string, timeout time.Duration) *Client {
	return &Client{
		base: "http://" + host,
		user: user,
		pass: password,
		http: &http.Client{Timeout: timeout},
	}
}

func ParseTime(s string) (time.Time, error) {
	return time.Parse(TimeLayout, s)
}

func (c *Client) get(ctx context.Context, path string, q url.Values) (*http.Response, error) {
	u := c.base + path
	if len(q) > 0 {
		u += "?" + q.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(c.user, c.pass)
	return c.http.Do(req)
}

func (c *Client) List(ctx context.Context, from time.Time, max int) ([]Recording, error) {
	q := url.Values{}
	q.Set("Datafrom", from.UTC().Format(TimeLayout))
	q.Set("Maxmatches", strconv.Itoa(max))
	resp, err := c.get(ctx, "/cgi-bin/get_mp4_list", q)
	if err != nil {
		return nil, fmt.Errorf("get_mp4_list: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get_mp4_list: unexpected status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if err != nil {
		return nil, fmt.Errorf("get_mp4_list: %w", err)
	}
	return parseList(string(body))
}

func parseList(body string) ([]Recording, error) {
	var recs []Recording
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		lower := strings.ToLower(line)
		if strings.HasPrefix(lower, "mp4filenum=") {
			continue
		}
		if strings.HasPrefix(lower, "errorno=") {
			if strings.TrimPrefix(lower, "errorno=") == "3" {
				return nil, ErrBusy
			}
			return nil, fmt.Errorf("get_mp4_list: camera returned %s", line)
		}
		fields := strings.Split(line, ",")
		if len(fields) != 3 {
			return nil, fmt.Errorf("get_mp4_list: unexpected line %q", line)
		}
		start, err := ParseTime(fields[1])
		if err != nil {
			return nil, fmt.Errorf("get_mp4_list: start time of %q: %w", fields[0], err)
		}
		end, err := ParseTime(fields[2])
		if err != nil {
			return nil, fmt.Errorf("get_mp4_list: end time of %q: %w", fields[0], err)
		}
		recs = append(recs, Recording{Name: fields[0], Start: start, End: end, EndRaw: fields[2]})
	}
	sort.SliceStable(recs, func(i, j int) bool { return recs[i].End.Before(recs[j].End) })
	return recs, nil
}

type body struct {
	io.Reader
	closer io.Closer
}

func (b body) Close() error { return b.closer.Close() }

func (c *Client) Download(ctx context.Context, name string) (io.ReadCloser, error) {
	q := url.Values{}
	q.Set("FileName", name)
	resp, err := c.get(ctx, "/cgi-bin/get_mp4_file", q)
	if err != nil {
		return nil, fmt.Errorf("get_mp4_file %s: %w", name, err)
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("get_mp4_file %s: unexpected status %d", name, resp.StatusCode)
	}
	br := bufio.NewReaderSize(resp.Body, 64<<10)
	head, _ := br.Peek(8)
	if len(head) == 8 && bytes.Equal(head[4:8], []byte("ftyp")) {
		return body{Reader: br, closer: resp.Body}, nil
	}
	defer resp.Body.Close()
	msg, _ := io.ReadAll(io.LimitReader(br, 256))
	text := strings.TrimSpace(string(msg))
	if strings.EqualFold(text, "ErrorNo=3") {
		return nil, ErrBusy
	}
	return nil, fmt.Errorf("get_mp4_file %s: not an mp4 response: %q", name, text)
}
