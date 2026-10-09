package item

import (
	"io"

	"github.com/ArtemChadaev/Auction/cmd/mimeType"
	"github.com/gabriel-vasile/mimetype"
)

var (
	errInvalidExtension = mimeType.ErrInvalidExtension
	errUnsupportedType  = mimeType.ErrUnsupportedType
	errFileSizeExceeded = mimeType.ErrFileSizeExceeded
)

func MaxSizeForType(t itemType) int64 {
	return mimeType.MaxSizeForType(mimeType.ItemType(t))
}

func validateSample(r io.Reader, filename string, fileSize int64) (itemType, *mimetype.MIME, error) {
	t, m, err := mimeType.ValidateSample(r, filename, fileSize)
	return itemType(t), m, err
}
