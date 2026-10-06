package item

import (
	"encoding/json"
	"net/textproto"
	"time"
	"uuid"
)

type itemType string

var (
	Image    itemType = "image"
	Model3D  itemType = "3d"
	Audio    itemType = "audio"
	Video    itemType = "video"
	Document itemType = "document"
	Archive  itemType = "archive"
)

// Каноничное расширение от mtype.Extension() -> разрешенные синонимы
var extensionAliases = map[string][]string{
	// Images
	".jpg":  {".jpeg", ".jpe", ".jfif"},
	".tiff": {".tif"},

	// Video
	".mp4": {".m4v"},
	".mov": {".qt"},
	".mpg": {".mpeg", ".mpe", ".m2v"},
	".ts":  {".mts", ".m2ts"},

	// Audio
	".ogg": {".oga", ".opus"},
	".aif": {".aiff", ".aifc"},

	// Documents & Text
	".doc": {".dot"},
	".xls": {".xla", ".xlt"},
	".ppt": {".pot", ".pps"},
	".txt": {".csv", ".tsv", ".log", ".conf", ".ini", ".env"},

	// Archives & 3D
	".tar":  {".gtar"},
	".gz":   {".tgz", ".prproj"},
	".step": {".stp"},
}

type Item struct {
	ID        int64           `db:"id" json:"id"`
	CreatorID uuid.UUID       `db:"creator_id" json:"creator_id"`
	OwnerID   uuid.UUID       `db:"owner_id" json:"owner_id"`
	S3ID      uuid.UUID       `db:"s3_id" json:"-"`
	Type      itemType        `db:"type" json:"type"`
	MetaData  json.RawMessage `db:"metadata" json:"metadata"`
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

type headerMetadata struct {
	MimeType string               `json:"mime_type"`
	Filename string               `json:"filename"`
	Size     int64                `json:"size"`
	Header   textproto.MIMEHeader `json:"header"`
}
type ImageMetadata struct {
	headerMetadata
	Width    int       `json:"width"`
	Height   int       `json:"height"`
	BlurHash string    `json:"blur_hash"`
	Preview  uuid.UUID `json:"preview"`
}
type Model3DMetadata struct {
	headerMetadata
	PolygonCount  int    `json:"polygon_count"`
	VertexCount   int    `json:"vertex_count"`
	HasTexture    bool   `json:"has_texture"`
	HasRigging    bool   `json:"has_rigging"`
	HasAnimations bool   `json:"has_animations"`
	RenderEngine  string `json:"render_engine"`
}
type AudioMetadata struct {
	headerMetadata
	Duration     time.Duration `json:"duration"`
	SampleRateHZ int           `json:"sample_rate_hz"`
	BitrateKbps  int           `json:"bitrate_kbps"`
	Channels     int           `json:"channels"`
	WaveformData []float32     `json:"waveform_data"`
	PreviewImage uuid.UUID     `json:"preview_image"`
	PreviewAudio uuid.UUID     `json:"preview_audio"`
}
type VideoMetadata struct {
	headerMetadata
	Duration     time.Duration `json:"duration"`
	Width        int           `json:"width"`
	Height       int           `json:"height"`
	BlurHash     string        `json:"blur_hash"`
	PreviewImage uuid.UUID     `json:"preview_image"`
	PreviewVideo uuid.UUID     `json:"preview_video"`
}
type DocumentMetadata struct {
	headerMetadata
	PageCount int `json:"page_count"`
}
type ArchiveMetadata struct {
	headerMetadata
	FileCount             int      `json:"file_count"`
	UncompressedSizeBytes int64    `json:"uncompressed_size_bytes"`
	Tree                  []string `json:"tree"`
}
