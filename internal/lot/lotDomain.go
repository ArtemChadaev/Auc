package lot

import (
	"time"
	"uuid"

	"github.com/ArtemChadaev/Auction/cmd/apperr"
)

// Lot status constants matching database check constraint
const (
	StatusDraft         = "draft"
	StatusPendingReview = "pending_review"
	StatusApproved      = "approved"
	StatusActive        = "active"
	StatusRejected      = "rejected"
	StatusFinished      = "finished"
	StatusCancelled     = "cancelled"
)

// MediaType specifies supported media format in lot description
type MediaType string

const (
	MediaTypeImage MediaType = "image"
	MediaTypeVideo MediaType = "video"
	MediaType3D    MediaType = "3d"
)

// MediaItem represents an uploaded photo or video associated with lot description.
// ID is a UUID pointing to the S3 stored item.
type MediaItem struct {
	ID          uuid.UUID  `json:"id"`
	Type        MediaType  `json:"type"`
	BlurHash    string     `json:"blur_hash,omitempty"`
	Width       int        `json:"width,omitempty"`
	Height      int        `json:"height,omitempty"`
	DurationSec float64    `json:"duration_sec,omitempty"`
	ThumbnailID *uuid.UUID `json:"thumbnail_id,omitempty"`
	Order       int        `json:"order"`
	Caption     string     `json:"caption,omitempty"`
}

// LotDescription contains full rich description, media attachments and technical specs.
type LotDescription struct {
	Text       string            `json:"text"`
	ShortText  string            `json:"short_text,omitempty"`
	Media      []MediaItem       `json:"media"`
	Attributes map[string]string `json:"attributes,omitempty"`
}

// SellerInfo contains public information about lot seller.
type SellerInfo struct {
	ID       uuid.UUID  `json:"id"`
	Name     string     `json:"name"`
	AvatarID *uuid.UUID `json:"avatar_id,omitempty"`
	Rating   float64    `json:"rating,omitempty"`
}

// WinnerInfo contains public information about auction winner once finished.
type WinnerInfo struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

// ItemSummary contains quick preview metadata of the underlying item being auctioned.
type ItemSummary struct {
	ID       int64     `json:"id"`
	Type     string    `json:"type"`
	Preview  uuid.UUID `json:"preview"`
	BlurHash string    `json:"blur_hash,omitempty"`
}

// Lot represents the complete auction lot entity returned to clients.
type Lot struct {
	ID           int64          `json:"id"`
	SellerID     uuid.UUID      `json:"seller_id"`
	Seller       SellerInfo     `json:"seller"`
	ItemID       int64          `json:"item_id"`
	ItemSummary  *ItemSummary   `json:"item_summary,omitempty"`
	Status       string         `json:"status"`
	Category     string         `json:"category"`
	Title        string         `json:"title"`
	Description  LotDescription `json:"description"`
	StartAmount  int64          `json:"start_amount"`
	CurrentBid   int64          `json:"current_bid"`
	MinStep      int64          `json:"min_step"`
	FinishAmount *int64         `json:"finish_amount,omitempty"`
	BidsCount    int            `json:"bids_count"`
	Winner       *WinnerInfo    `json:"winner,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
	StartedAt    *time.Time     `json:"started_at,omitempty"`
	EndsAt       *time.Time     `json:"ends_at,omitempty"`
	FinishedAt   *time.Time     `json:"finished_at,omitempty"`
}

// LotFilter describes URL query parameters for filtering and sorting lots.
type LotFilter struct {
	Query    string
	Category string
	Status   string
	MinPrice *int64
	MaxPrice *int64
	Sort     string
	Page     int
	Limit    int
}

// LotListResponse represents paginated response returned by GET /
type LotListResponse struct {
	Items      []Lot `json:"items"`
	Total      int   `json:"total"`
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	TotalPages int   `json:"total_pages"`
}

var (
	ErrInvalidLotID = apperr.NewAppErrorString("invalid lot id", -4)
	ErrLotNotFound  = apperr.NewAppErrorString("lot not found", -4)
)
