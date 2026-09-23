package item

import (
	"encoding/json"
	"time"
	"uuid"
)

type Type string

var (
	Image    Type = "image"
	Model3D  Type = "3d"
	Audio    Type = "audio"
	Video    Type = "video"
	Document Type = "document"
	Archive  Type = "archive"
)

type Item struct {
	ID        int64           `db:"id" json:"id"`
	CreatorID uuid.UUID       `db:"creator_id" json:"creator_id"`
	OwnerID   uuid.UUID       `db:"owner_id" json:"owner_id"`
	Key       string          `db:"key" json:"key"`
	Type      Type            `db:"type" json:"type"`
	MimeType  string          `db:"mime_type" json:"mime_type"`
	Size      int64           `db:"file_size_bytes" json:"size"`
	Meta      json.RawMessage `db:"metadata" json:"metadata"`
	CreatedAt time.Time       `db:"created_at" json:"created_at"`
}

type HistoryItem struct {
	ID          int64     `db:"id" json:"id"`
	ItemID      int64     `db:"item_id" json:"item_id"`
	OldUser     uuid.UUID `db:"old_user" json:"old_user"`
	NewUser     uuid.UUID `db:"new_user" json:"new_user"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	LotID       int64     `db:"lot_id" json:"lot_id"`
	Description string    `db:"description" json:"description"`
}

// Metadata Структуры для metadata items
type Metadata interface {
	ImageMetadata | Model3DMetadata | AudioMetadata | VideoMetadata | DocumentMetadata | ArchiveMetadata
}
type ImageMetadata struct {
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	BlurHash string `json:"blur_hash"`
}

type Model3DMetadata struct {
	PolygonCount  int    `json:"polygon_count"`
	VertexCount   int    `json:"vertex_count"`
	HasTexture    bool   `json:"has_texture"`
	HasRigging    bool   `json:"has_rigging"`
	HasAnimations bool   `json:"has_animations"`
	RenderEngine  string `json:"render_engine"`
}

type AudioMetadata struct {
	Duration     time.Duration `json:"duration"`
	SampleRateHZ int           `json:"sample_rate_hz"`
	BitrateKbps  int           `json:"bitrate_kbps"`
	Channels     int           `json:"channels"`
	WaveformData []float32
}
type VideoMetadata struct {
	Duration time.Duration `json:"duration"`
	Width    int           `json:"width"`
	Height   int           `json:"height"`
}
type DocumentMetadata struct {
	PageCount int    `json:"page_count"`
	Language  string `json:"language"`
}
type ArchiveMetadata struct {
	FileCount             int      `json:"file_count"`
	UncompressedSizeBytes int      `json:"uncompressed_size_bytes"`
	FileTree              []string `json:"file_tree"`
}
