package storageS3

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"time"
	"uuid"

	"github.com/ArtemChadaev/Auction/cmd/apperr"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
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
		return nil, fmt.Errorf("failed to load config: %w", apperr.NewAppError(err, 4))
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
		return nil, fmt.Errorf("failed to download: %w", apperr.NewAppError(err, 4))
	}
	return result.Body, nil
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
// Полезно вызывать перед генерацией URL, передавая хеш файла:
// если файл уже есть, загружать его повторно не нужно (дедупликация).
func (c *Client) Exists(ctx context.Context, key string) (bool, error) {
	_, err := c.s3Client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		var notFound *types.NotFound
		var noSuchKey *types.NoSuchKey
		if errors.As(err, &notFound) || errors.As(err, &noSuchKey) {
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
		return "", apperr.NewAppErrorString("0 < size <= 10 gb", -4)
	}

	presignClient := s3.NewPresignClient(c.s3Client)

	// ВАЖНО: В PutObjectInput передаем только Bucket и Key.
	// Если включить ChecksumSHA256 или ContentLength, AWS SDK добавит их в X-Amz-SignedHeaders.
	// Supabase S3 Gateway и сторонние клиенты (браузер fetch, JetBrains HTTP Client)
	// не поддерживают x-amz-checksum-* в SigV4 или блокируют заголовок Content-Length,
	// что приводит к ошибке 403 SignatureDoesNotMatch.
	input := &s3.PutObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String("tmp/" + key.String()),
	}

	request, err := presignClient.PresignPutObject(ctx, input, s3.WithPresignExpires(DefaultUploadLifetime))
	if err != nil {
		return "", fmt.Errorf("failed to presign upload url: %w", apperr.NewAppError(err, 4))
	}

	slog.DebugContext(ctx, "request GetURLForUpload", slog.Any("request", request))

	return request.URL, nil
}
