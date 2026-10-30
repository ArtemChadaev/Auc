package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/ArtemChadaev/Auction/cmd/apperr"
	"github.com/ArtemChadaev/Auction/cmd/storageS3"
	"github.com/ArtemChadaev/Auction/cmd/valkey"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ViolationRecord представляет запись о нарушении лимита запросов.
type ViolationRecord struct {
	ID            int64     `db:"id" json:"id,omitempty"`
	Identifier    string    `db:"identifier" json:"identifier"`
	IP            string    `db:"ip" json:"ip"`
	UserAgent     string    `db:"user_agent" json:"user_agent"`
	Endpoint      string    `db:"endpoint" json:"endpoint"`
	RequestCount  int64     `db:"request_count" json:"request_count"`
	LimitCount    int       `db:"limit_count" json:"limit_count"`
	ViolationType string    `db:"violation_type" json:"violation_type"` // "global", "endpoint:...", "ip_safety"
	CreatedAt     time.Time `db:"created_at" json:"created_at"`
}

// ViolationExporter отвечает за регулярную выгрузку накопленных нарушений из буфера Redis в PostgreSQL и S3.
type ViolationExporter struct {
	vk        *valkey.Client
	pool      *pgxpool.Pool
	s3Client  *storageS3.Client
	interval  time.Duration
	batchSize int
}

// NewViolationExporter создает новый экземпляр экспортера нарушений.
func NewViolationExporter(
	vk *valkey.Client,
	pool *pgxpool.Pool,
	s3Client *storageS3.Client,
	interval time.Duration,
) *ViolationExporter {
	if interval <= 0 {
		interval = 10 * time.Minute
	}
	return &ViolationExporter{
		vk:        vk,
		pool:      pool,
		s3Client:  s3Client,
		interval:  interval,
		batchSize: 500,
	}
}

// InitSchema проверяет и создаёт таблицу rate_limit_violations в PostgreSQL при необходимости.
func (exp *ViolationExporter) InitSchema(ctx context.Context) error {
	if exp.pool == nil {
		return nil
	}
	_, err := exp.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS rate_limit_violations (
			id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
			identifier TEXT NOT NULL,
			ip TEXT NOT NULL,
			user_agent TEXT,
			endpoint TEXT NOT NULL,
			request_count INT NOT NULL,
			limit_count INT NOT NULL,
			violation_type TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);
		CREATE INDEX IF NOT EXISTS idx_violations_identifier ON rate_limit_violations(identifier);
		CREATE INDEX IF NOT EXISTS idx_violations_created_at ON rate_limit_violations(created_at);
	`)
	if err != nil {
		return fmt.Errorf("ViolationExporter.InitSchema: %w", err)
	}
	return nil
}

// Start запускает цикл периодической выгрузки нарушений в фоне.
func (exp *ViolationExporter) Start(ctx context.Context) {
	if err := exp.InitSchema(ctx); err != nil {
		apperr.Log(ctx, "ViolationExporter.Start.InitSchema", err)
	}

	ticker := time.NewTicker(exp.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Выгружаем остаток очереди при штатном завершении работы сервера
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			_ = exp.Flush(shutdownCtx)
			cancel()
			return
		case <-ticker.C:
			if err := exp.Flush(ctx); err != nil {
				apperr.Log(ctx, "ViolationExporter.Flush", err)
			}
		}
	}
}

// Flush выгружает пачку нарушений из буфера Redis в PostgreSQL и архивный S3 бакет.
func (exp *ViolationExporter) Flush(ctx context.Context) error {
	if exp.vk == nil || exp.vk.Raw() == nil {
		return nil
	}

	rdb := exp.vk.Raw()

	// Атомарно извлекаем пачку нарушений из начала очереди Redis
	pipe := rdb.TxPipeline()
	lrangeCmd := pipe.LRange(ctx, "ratelimit:violations:queue", 0, int64(exp.batchSize-1))
	pipe.LTrim(ctx, "ratelimit:violations:queue", int64(exp.batchSize), -1)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("ViolationExporter.Flush.Redis: %w", err)
	}

	rawItems := lrangeCmd.Val()
	if len(rawItems) == 0 {
		return nil
	}

	var records []ViolationRecord
	for _, item := range rawItems {
		var rec ViolationRecord
		if err := json.Unmarshal([]byte(item), &rec); err == nil {
			records = append(records, rec)
		}
	}

	if len(records) == 0 {
		return nil
	}

	// 1. Выгрузка в PostgreSQL таблицу rate_limit_violations
	if exp.pool != nil {
		batch := &pgx.Batch{}
		for _, rec := range records {
			batch.Queue(`
				INSERT INTO rate_limit_violations (identifier, ip, user_agent, endpoint, request_count, limit_count, violation_type, created_at)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			`, rec.Identifier, rec.IP, rec.UserAgent, rec.Endpoint, rec.RequestCount, rec.LimitCount, rec.ViolationType, rec.CreatedAt)
		}
		br := exp.pool.SendBatch(ctx, batch)
		if err := br.Close(); err != nil {
			apperr.Log(ctx, "ViolationExporter.Flush.pgxBatch", err)
		}
	}

	// 2. Выгрузка архивного JSON батча в S3
	var s3Key string
	if exp.s3Client != nil {
		now := time.Now().UTC()
		batchJSON, err := json.MarshalIndent(records, "", "  ")
		if err == nil {
			s3Key = fmt.Sprintf("audit/violations/%s/violations_%s.json",
				now.Format("2006/01/02"),
				now.Format("20060102_150405"),
			)
			if upErr := exp.s3Client.Upload(ctx, s3Key, bytes.NewReader(batchJSON), "application/json"); upErr != nil {
				apperr.Log(ctx, "ViolationExporter.Flush.s3Upload", upErr, "key", s3Key)
			}
		}
	}

	slog.Info("violation_exporter_flush_success",
		"exported_records", len(records),
		"db_saved", exp.pool != nil,
		"s3_key", s3Key,
	)

	return nil
}

// GetRecentViolations возвращает последние записи нарушений из базы данных для мониторинга/аудита.
func (exp *ViolationExporter) GetRecentViolations(ctx context.Context, limit int) ([]ViolationRecord, error) {
	if exp.pool == nil {
		return nil, nil
	}
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	rows, err := exp.pool.Query(ctx, `
		SELECT id, identifier, ip, user_agent, endpoint, request_count, limit_count, violation_type, created_at
		FROM rate_limit_violations
		ORDER BY created_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("ViolationExporter.GetRecentViolations: %w", err)
	}
	defer rows.Close()

	var results []ViolationRecord
	for rows.Next() {
		var r ViolationRecord
		if err := rows.Scan(&r.ID, &r.Identifier, &r.IP, &r.UserAgent, &r.Endpoint, &r.RequestCount, &r.LimitCount, &r.ViolationType, &r.CreatedAt); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, nil
}
