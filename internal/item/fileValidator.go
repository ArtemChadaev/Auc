package item

import (
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
)

const (
	MaxArchiveSize  int64 = 5 << 30   // 5 ГБ (только архивы до 5 ГБ)
	MaxVideoSize    int64 = 1 << 30   // 1 ГБ (видео до 1 ГБ)
	MaxAudioSize    int64 = 500 << 20 // 500 МБ
	MaxModel3DSize  int64 = 500 << 20 // 500 МБ
	MaxDocumentSize int64 = 200 << 20 // 200 МБ
	MaxImageSize    int64 = 100 << 20 // 100 МБ
)

func MaxSizeForType(t itemType) int64 {
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

// validateFileExtension: если расширение файла не сходится с содержимым -> false
func validateFileExtension(filename string, mtype *mimetype.MIME) bool {
	userExt := strings.ToLower(filepath.Ext(filename))
	if userExt == "" {
		return false
	}

	ext := mtype.Extension()
	if ext == userExt {
		return true
	}

	if aliases, ok := extensionAliases[ext]; ok {
		if slices.Contains(aliases, userExt) {
			return true
		}
	}
	return false
}

// fileType возвращает краткий тип файла
func fileType(filename string, mtype *mimetype.MIME) itemType {
	mimeStr := mtype.String()
	userExt := strings.ToLower(filepath.Ext(filename))

	if strings.HasPrefix(mimeStr, "image/") {
		return Image
	}

	if strings.HasPrefix(mimeStr, "audio/") {
		return Audio
	}

	if strings.HasPrefix(mimeStr, "video/") {
		return Video
	}

	// 4. 3D-МОДЕЛИ
	if strings.HasPrefix(mimeStr, "model/") ||
		mimeStr == "application/sla" || // устаревший MIME для STL
		mimeStr == "application/x-tgif" ||
		userExt == ".obj" || userExt == ".fbx" || userExt == ".step" || userExt == ".stp" {
		return Model3D
	}

	// 5. ДОКУМЕНТЫ
	if mimeStr == "application/pdf" ||
		mimeStr == "application/rtf" ||
		mimeStr == "application/epub+zip" ||
		mtype.Is("application/msword") ||
		strings.Contains(mimeStr, "openxmlformats-officedocument") ||
		strings.Contains(mimeStr, "opendocument") ||
		userExt == ".csv" || userExt == ".txt" || userExt == ".tsv" || userExt == ".md" {
		return Document
	}

	// 6. АРХИВЫ И ОБРАЗЫ ДИСКОВ
	if mtype.Is("application/zip") ||
		mtype.Is("application/x-tar") ||
		mimeStr == "application/x-7z-compressed" ||
		mimeStr == "application/vnd.rar" ||
		mimeStr == "application/x-rar-compressed" ||
		mimeStr == "application/gzip" ||
		mimeStr == "application/x-bzip2" ||
		mimeStr == "application/x-xz" ||
		mimeStr == "application/x-iso9660-image" ||
		mimeStr == "application/zstd" {
		return Archive
	}
	return ""
}

// validateSample проверяет начальные байты (каплю) файла на соответствие расширению и лимитам размера
func validateSample(r io.Reader, filename string, fileSize int64) (itemType, *mimetype.MIME, error) {
	mtype, err := mimetype.DetectReader(r)
	if err != nil {
		return "", nil, fmt.Errorf("fileValidator.validateSample: %w", apperr.NewAppError(err, -4))
	}

	if !validateFileExtension(filename, mtype) {
		return "", mtype, ErrInvalidExtension
	}

	iType := fileType(filename, mtype)
	if iType == "" {
		return "", mtype, ErrUnsupportedType
	}

	maxAllowed := MaxSizeForType(iType)
	if fileSize > maxAllowed {
		return iType, mtype, fmt.Errorf("fileValidator.validateSample: %w", ErrFileSizeExceeded)
	}

	return iType, mtype, nil
}
