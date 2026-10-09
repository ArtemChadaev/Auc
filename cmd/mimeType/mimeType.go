package mimeType

import (
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"slices"
	"strings"

	"github.com/ArtemChadaev/Auction/cmd/apperr"
	"github.com/gabriel-vasile/mimetype"
)

var (
	ErrInvalidExtension = apperr.NewAppErrorString("file extension does not match content", -4)
	ErrUnsupportedType  = apperr.NewAppErrorString("unsupported file type", -4)
	ErrFileSizeExceeded = apperr.NewAppErrorString("file size exceeds maximum allowed for this type", -4)
	ErrNotImage         = apperr.NewAppErrorString("file is not an image", -4)
)

const (
	MaxArchiveSize  int64 = 5 << 30   // 5 ГБ (только архивы до 5 ГБ)
	MaxVideoSize    int64 = 1 << 30   // 1 ГБ (видео до 1 ГБ)
	MaxAudioSize    int64 = 500 << 20 // 500 МБ
	MaxModel3DSize  int64 = 500 << 20 // 500 МБ
	MaxDocumentSize int64 = 200 << 20 // 200 МБ
	MaxImageSize    int64 = 100 << 20 // 100 МБ
	MaxPhotoSize    int64 = 5 << 20   // 5 МБ (фотографии пользователей)
)

type ItemType string

const (
	Image    ItemType = "image"
	Model3D  ItemType = "3d"
	Audio    ItemType = "audio"
	Video    ItemType = "video"
	Document ItemType = "document"
	Archive  ItemType = "archive"
)

// ExtensionAliases - каноническое расширение -> разрешенные синонимы
var ExtensionAliases = map[string][]string{
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

func MaxSizeForType(t ItemType) int64 {
	switch t {
	case Archive:
		return MaxArchiveSize
	case Video:
		return MaxVideoSize
	case Audio:
		return MaxAudioSize
	case Model3D:
		return MaxModel3DSize
	case Document:
		return MaxDocumentSize
	case Image:
		return MaxImageSize
	default:
		return MaxVideoSize
	}
}

// MatchExtension проверяет, соответствует ли расширение файла MIME-типу
func MatchExtension(filename string, mtype *mimetype.MIME) bool {
	userExt := strings.ToLower(filepath.Ext(filename))
	canonicalExt := mtype.Extension()

	if strings.EqualFold(userExt, canonicalExt) {
		return true
	}
	if aliases, ok := ExtensionAliases[canonicalExt]; ok {
		if slices.Contains(aliases, userExt) {
			return true
		}
	}
	if mtype.String() == "text/plain" && (userExt == ".txt" || userExt == ".csv" || userExt == ".md") {
		return true
	}
	return false
}

// ValidateSample проверяет первые 512 байт файла (каплю), сверяет MIME-тип с расширением
// и контролирует допустимый размер файла для данного типа айтема.
func ValidateSample(r io.Reader, filename string, fileSize int64) (ItemType, *mimetype.MIME, error) {
	sample := make([]byte, 512)
	n, err := io.ReadFull(r, sample)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return "", nil, fmt.Errorf("mimeType.ValidateSample: %w", apperr.NewAppError(err, 4))
	}
	if n == 0 {
		return "", nil, ErrUnsupportedType
	}

	mtype := mimetype.Detect(sample[:n])
	if !MatchExtension(filename, mtype) {
		return "", nil, ErrInvalidExtension
	}

	userExt := strings.ToLower(filepath.Ext(filename))
	var iType ItemType
	switch {
	case strings.HasPrefix(mtype.String(), "image/"):
		iType = Image
	case strings.HasPrefix(mtype.String(), "video/"):
		iType = Video
	case strings.HasPrefix(mtype.String(), "audio/"):
		iType = Audio
	case mtype.String() == "model/gltf-binary" || mtype.String() == "model/gltf+json" || userExt == ".glb" || userExt == ".gltf":
		iType = Model3D
	case mtype.String() == "application/pdf" ||
		mtype.String() == "application/vnd.openxmlformats-officedocument.wordprocessingml.document" ||
		mtype.String() == "application/vnd.openxmlformats-officedocument.presentationml.presentation" ||
		mtype.String() == "text/plain":
		iType = Document
	case mtype.String() == "application/zip" ||
		mtype.String() == "application/x-tar" ||
		mtype.String() == "application/gzip" ||
		mtype.String() == "application/x-7z-compressed" ||
		mtype.String() == "application/vnd.rar":
		iType = Archive
	default:
		return "", nil, ErrUnsupportedType
	}

	maxAllowed := MaxSizeForType(iType)
	if fileSize > maxAllowed {
		return "", nil, ErrFileSizeExceeded
	}

	return iType, mtype, nil
}

// ValidatePhoto проверяет, что файл является простым растровым изображением (не SVG),
// расширение файла соответствует его содержимому, и размер не превышает лимит.
func ValidatePhoto(r io.Reader, filename string, fileSize int64) (*mimetype.MIME, error) {
	sample := make([]byte, 512)
	n, err := io.ReadFull(r, sample)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		return nil, fmt.Errorf("mimeType.ValidatePhoto: %w", apperr.NewAppError(err, 4))
	}
	if n == 0 {
		return nil, ErrUnsupportedType
	}

	mtype := mimetype.Detect(sample[:n])

	// Проверяем, что это чисто image/ (растровые фотографии, кроме svg)
	if !strings.HasPrefix(mtype.String(), "image/") || mtype.String() == "image/svg+xml" {
		return nil, ErrNotImage
	}

	// Проверяем соответствие расширения и содержимого
	if !MatchExtension(filename, mtype) {
		return nil, ErrInvalidExtension
	}

	if fileSize > MaxPhotoSize {
		return nil, ErrFileSizeExceeded
	}

	return mtype, nil
}
