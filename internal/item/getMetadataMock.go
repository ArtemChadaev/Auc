package item

import (
	"context"
	"encoding/json"
	"io"
	"time"
	"uuid"
)

// Мок-функции генерации метаданных без ошибок на основе структур из itemDomain.go

func mockImageMetadata(header headerMetadata) ImageMetadata {
	return ImageMetadata{
		headerMetadata: header,
		Width:          1920,
		Height:         1080,
		BlurHash:       "LEHLk~WB2yk8pyo0adR*.7kCMdnj",
		Preview:        uuid.New(),
	}
}

func mockModel3DMetadata(header headerMetadata) Model3DMetadata {
	return Model3DMetadata{
		headerMetadata: header,
		PolygonCount:   15000,
		VertexCount:    8000,
		HasTexture:     true,
		HasRigging:     false,
		HasAnimations:  false,
		RenderEngine:   "Standard",
	}
}

func mockAudioMetadata(header headerMetadata) AudioMetadata {
	return AudioMetadata{
		headerMetadata: header,
		Duration:       3*time.Minute + 30*time.Second,
		SampleRateHZ:   44100,
		BitrateKbps:    320,
		Channels:       2,
		WaveformData:   []float32{0.1, 0.4, 0.8, 0.6, 0.3, 0.2},
		PreviewImage:   uuid.New(),
		PreviewAudio:   uuid.New(),
	}
}

func mockVideoMetadata(header headerMetadata) VideoMetadata {
	return VideoMetadata{
		headerMetadata: header,
		Duration:       1*time.Minute + 45*time.Second,
		Width:          1920,
		Height:         1080,
		BlurHash:       "L6PZfSi_.AyE_3t7t7R**0o#DgR4",
		PreviewImage:   uuid.New(),
		PreviewVideo:   uuid.New(),
	}
}

func mockDocumentMetadata(header headerMetadata) DocumentMetadata {
	return DocumentMetadata{
		headerMetadata: header,
		PageCount:      5,
	}
}

func mockArchiveMetadata(header headerMetadata) ArchiveMetadata {
	return ArchiveMetadata{
		headerMetadata:        header,
		FileCount:             10,
		UncompressedSizeBytes: header.Size * 2,
		Tree:                  []string{"file1.txt", "assets/image.png", "data.json"},
	}
}

// createItemMetadata генерирует моковые метаданные без ошибок
func createItemMetadata(ctx context.Context, iType itemType, mType string, hmData headerMetadata, pr *io.PipeReader) (json.RawMessage, error) {
	if pr != nil {
		_, _ = io.Copy(io.Discard, pr)
	}

	var meta any
	switch iType {
	case Image:
		meta = mockImageMetadata(hmData)
	case Model3D:
		meta = mockModel3DMetadata(hmData)
	case Audio:
		meta = mockAudioMetadata(hmData)
	case Video:
		meta = mockVideoMetadata(hmData)
	case Document:
		meta = mockDocumentMetadata(hmData)
	case Archive:
		meta = mockArchiveMetadata(hmData)
	default:
		meta = mockImageMetadata(hmData)
	}

	data, _ := json.Marshal(meta)
	return data, nil
}
