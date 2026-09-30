package storageS3

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"time"

	"github.com/ArtemChadaev/Auction/cmd/apperr"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type Client struct {
	s3Client *s3.Client
	bucket   string
}
type Config struct {
	Endpoint  string
	Region    string
	AccessKey string
	SecretKey string
	Bucket    string
}

func NewClient(ctx context.Context, cfg Config) (*Client, error) {
	s3Config, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(cfg.Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, "")),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to load config(%w): %s", apperr.ErrError, err)
	}

	client := s3.NewFromConfig(s3Config, func(o *s3.Options) {
		if cfg.Endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
		}
		o.UsePathStyle = true
	})

	return &Client{
		s3Client: client,
		bucket:   cfg.Bucket,
	}, nil
}

func (c *Client) Upload(ctx context.Context, key string, r io.Reader, contentType string) error {
	_, err := c.s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(c.bucket),
		Key:         aws.String(key),
		Body:        r,
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return fmt.Errorf("failed to upload(%w), key(%s): %s", apperr.ErrWarn, key, err)
	}
	return nil
}

func (c *Client) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	result, err := c.s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to download(%w), key(%s): %s", apperr.ErrWarn, key, err)
	}
	return result.Body, nil
}

func (c *Client) Delete(ctx context.Context, key string) error {
	_, err := c.s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to delete(%w), key(%s): %s", apperr.ErrWarn, key, err)
	}
	return nil
}

func (c *Client) GetURL(ctx context.Context, key string, lifetime time.Duration, origName string) (string, error) {
	presignClient := s3.NewPresignClient(c.s3Client)
	contentDisposition := fmt.Sprintf("attachment; filename*=UTF-8''%s", url.PathEscape(origName))

	request, err := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket:                     aws.String(c.bucket),
		Key:                        aws.String(key),
		ResponseContentDisposition: aws.String(contentDisposition),
	}, s3.WithPresignExpires(lifetime))
	if err != nil {
		return "", fmt.Errorf("failed to presign(%w), key(%s): %s", apperr.ErrWarn, key, err)
	}
	return request.URL, nil
}
