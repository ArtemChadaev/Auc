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
	errInvalidExtension = apperr.NewAppErrorString("file extension does not match content", -4)
	errUnsupportedType  = apperr.NewAppErrorString("unsupported file type", -4)
	errFileSizeExceeded = apperr.NewAppErrorString("file size exceeds maximum allowed for this type", -4)
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

// validateSample проверяет первые 512 байт файла (каплю), сверяет MIME-тип с расширением
// и контролирует допустимый размер файла для данного типа.
func validateSample(r io.Reader, filename string, fileSize int64) (itemType, *mimetype.MIME, error) {
	sample := make([]byte, 512)
	n, err := io.ReadFull(r, sample)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return "", nil, fmt.Errorf("fileValidator.validateSample: %w", apperr.NewAppError(err, 4))
	}
	if n == 0 {
		return "", nil, errUnsupportedType
	}

	mtype := mimetype.Detect(sample[:n])
	userExt := strings.ToLower(filepath.Ext(filename))
	canonicalExt := mtype.Extension()

	// Проверяем соответствие расширения файла и MIME-типа
	validExt := false
	if strings.EqualFold(userExt, canonicalExt) {
		validExt = true
	} else if aliases, ok := extensionAliases[canonicalExt]; ok {
		if slices.Contains(aliases, userExt) {
			validExt = true
		}
	}

	// Особый случай для plain text и svg
	if !validExt && mtype.String() == "text/plain" && (userExt == ".txt" || userExt == ".csv" || userExt == ".md") {
		validExt = true
	}

	if !validExt {
		return "", nil, errInvalidExtension
	}

	// Определяем тип элемента и проверяем лимиты размера
	var iType itemType
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
		return "", nil, errUnsupportedType
	}

	maxAllowed := MaxSizeForType(iType)
	if fileSize > maxAllowed {
		return "", nil, errFileSizeExceeded
	}

	return iType, mtype, nil
}
