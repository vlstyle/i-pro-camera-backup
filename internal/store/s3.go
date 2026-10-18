package store

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
)

type S3 struct {
	client   *s3.Client
	uploader *manager.Uploader
	bucket   string
	prefix   string
	camera   string
}

func NewS3(ctx context.Context, bucket, prefix, endpoint, camera string) (*S3, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		if endpoint != "" {
			o.BaseEndpoint = aws.String(endpoint)
			o.UsePathStyle = true
		}
	})
	return &S3{
		client:   client,
		uploader: manager.NewUploader(client),
		bucket:   bucket,
		prefix:   prefix,
		camera:   camera,
	}, nil
}

func (s *S3) Acquire(ctx context.Context) (func(), error) {
	return func() {}, nil
}

func (s *S3) PutVideo(ctx context.Context, key string, body io.Reader) error {
	_, err := s.uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(s.prefix + key),
		Body:        body,
		ContentType: aws.String("video/mp4"),
	})
	return err
}

func (s *S3) LoadCursor(ctx context.Context) (Cursor, string, error) {
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(s.prefix + cursorKey(s.camera)),
	})
	if err != nil {
		var nsk *types.NoSuchKey
		if errors.As(err, &nsk) {
			return Cursor{}, "", ErrNoCursor
		}
		return Cursor{}, "", err
	}
	defer out.Body.Close()
	data, err := io.ReadAll(io.LimitReader(out.Body, 1<<20))
	if err != nil {
		return Cursor{}, "", err
	}
	var c Cursor
	if err := json.Unmarshal(data, &c); err != nil {
		return Cursor{}, "", fmt.Errorf("invalid cursor object: %w", err)
	}
	return c, aws.ToString(out.ETag), nil
}

func (s *S3) SaveCursor(ctx context.Context, c Cursor, version string) (string, error) {
	data, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	in := &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(s.prefix + cursorKey(s.camera)),
		Body:        bytes.NewReader(data),
		ContentType: aws.String("application/json"),
	}
	if version == "" {
		in.IfNoneMatch = aws.String("*")
	} else {
		in.IfMatch = aws.String(version)
	}
	out, err := s.client.PutObject(ctx, in)
	if err != nil {
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) {
			switch apiErr.ErrorCode() {
			case "PreconditionFailed", "ConditionalRequestConflict":
				return "", ErrCursorConflict
			}
		}
		return "", err
	}
	return aws.ToString(out.ETag), nil
}
