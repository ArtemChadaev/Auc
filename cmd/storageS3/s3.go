package storageS3

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"
	"uuid"

	"github.com/ArtemChadaev/Auction/cmd/apperr"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
)

var ErrNotFound = apperr.NewAppErrorString("object not found", -4)

func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	var notFound *types.NotFound
	var noSuchKey *types.NoSuchKey
	if errors.As(err, &notFound) || errors.As(err, &noSuchKey) {
		return true
	}
	var respErr interface{ HTTPStatusCode() int }
	if errors.As(err, &respErr) && respErr.HTTPStatusCode() == 404 {
		return true
	}
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		code := apiErr.ErrorCode()
		if code == "NotFound" || code == "NoSuchKey" || code == "404" {
			return true
		}
	}
	if strings.Contains(err.Error(), "StatusCode: 404") || strings.Contains(err.Error(), "NotFound") {
		return true
	}
	return false
}

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
		return nil, fmt.Errorf("failed to load config: %w", apperr.NewAppError(err, 4))
	}

	client := s3.NewFromConfig(s3Config, func(o *s3.Options) {
		if cfg.Endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
		}
		o.UsePathStyle = true
		o.DisableLogOutputChecksumValidationSkipped = true
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
		return fmt.Errorf("failed to upload: %w", apperr.NewAppError(err, 4))
	}
	return nil
}

func (c *Client) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	result, err := c.s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		if isNotFound(err) {
			return nil, fmt.Errorf("failed to download: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("failed to download: %w", apperr.NewAppError(err, 4))
	}
	return result.Body, nil
}

func (c *Client) DownloadRange(ctx context.Context, key string, start, end int64) (io.ReadCloser, error) {
	rangeHeader := fmt.Sprintf("bytes=%d-%d", start, end)
	result, err := c.s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
		Range:  aws.String(rangeHeader),
	})
	if err != nil {
		if isNotFound(err) {
			return nil, fmt.Errorf("failed to download range: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("failed to download range: %w", apperr.NewAppError(err, 4))
	}
	return result.Body, nil
}

func (c *Client) Head(ctx context.Context, key string) (int64, error) {
	result, err := c.s3Client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		if isNotFound(err) {
			return 0, fmt.Errorf("failed to head object: %w", ErrNotFound)
		}
		return 0, fmt.Errorf("failed to head object: %w", apperr.NewAppError(err, 4))
	}
	if result.ContentLength == nil {
		return 0, nil
	}
	return *result.ContentLength, nil
}

func (c *Client) Move(ctx context.Context, srcKey, dstKey string) error {
	copySource := c.bucket + "/" + srcKey
	_, err := c.s3Client.CopyObject(ctx, &s3.CopyObjectInput{
		Bucket:     aws.String(c.bucket),
		Key:        aws.String(dstKey),
		CopySource: aws.String(copySource),
	})
	if err != nil {
		return fmt.Errorf("failed to copy object: %w", apperr.NewAppError(err, 4))
	}

	if err = c.Delete(ctx, srcKey); err != nil {
		return fmt.Errorf("failed to delete source object after copy: %w", err)
	}
	return nil
}

func (c *Client) Delete(ctx context.Context, key string) error {
	_, err := c.s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to delete: %w", apperr.NewAppError(err, 4))
	}
	return nil
}

// Exists проверяет, существует ли уже объект в бакете S3.
func (c *Client) Exists(ctx context.Context, key string) (bool, error) {
	_, err := c.s3Client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		if isNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check existence: %w", apperr.NewAppError(err, 4))
	}
	return true, nil
}

func (c *Client) GetURLForDownload(ctx context.Context, key string, lifetime time.Duration, origName string) (string, error) {
	presignClient := s3.NewPresignClient(c.s3Client)
	contentDisposition := fmt.Sprintf("attachment; filename*=UTF-8''%s", url.PathEscape(origName))

	request, err := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket:                     aws.String(c.bucket),
		Key:                        aws.String(key),
		ResponseContentDisposition: aws.String(contentDisposition),
	}, s3.WithPresignExpires(lifetime))
	if err != nil {
		return "", fmt.Errorf("failed to presign: %w", apperr.NewAppError(err, 4))
	}
	return request.URL, nil
}

// MaxUploadSize - максимальный допустимый размер файла для загрузки (5 ГБ).
const MaxUploadSize int64 = 5 << 30

// DefaultUploadLifetime - время жизни ссылки на загрузку по умолчанию (24 часа).
const DefaultUploadLifetime = 24 * time.Hour

// GetURLForUpload генерирует Presigned PUT URL для прямой загрузки файла клиентом в S3/Supabase/MinIO.
func (c *Client) GetURLForUpload(
	ctx context.Context,
	key uuid.UUID,
	fileSize int64,
) (string, error) {
	if fileSize > MaxUploadSize || fileSize <= 0 {
		return "", fmt.Errorf("invalid file size")
	}

	presignClient := s3.NewPresignClient(c.s3Client)
	s3Key := "tmp/" + key.String()

	input := &s3.PutObjectInput{
		Bucket:        aws.String(c.bucket),
		Key:           aws.String(s3Key),
		ContentLength: aws.Int64(fileSize),
	}

	req, err := presignClient.PresignPutObject(ctx, input, s3.WithPresignExpires(DefaultUploadLifetime))
	if err != nil {
		apperr.Log(ctx, "s3.GetURLForUpload", apperr.NewAppError(err, 4))
		return "", fmt.Errorf("s3.GetURLForUpload: %w", apperr.NewAppError(err, 4))
	}

	return req.URL, nil
}
