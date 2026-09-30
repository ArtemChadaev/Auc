package item

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"image"
	"io"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/ArtemChadaev/Auction/cmd/apperr"
	"github.com/bbrks/go-blurhash"
	"github.com/mholt/archives"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/qmuntal/gltf"
	"golang.org/x/image/draw"
	"golang.org/x/sync/errgroup"
	"gopkg.in/vansante/go-ffprobe.v2"
)

// TODO: Весь этот файл кроме Image сгенерирован (ао отдельности). Требуется полные тесты
// TODO: Потом на фронте подумать как выводить все (может сохранять обложку видео или аудио, первую страницу доков (или темы))
func getImageMetadata(ctx context.Context, header headerMetadata, mType string, r *io.PipeReader) (ImageMetadata, error) {
	var metadata ImageMetadata
	if mType == "image/svg+xml" {
		metadata = ImageMetadata{
			headerMetadata: header,
			Width:          0,
			Height:         0,
			BlurHash:       "",
		}
		return metadata, nil
	}
	g, gCtx := errgroup.WithContext(ctx)
	pr1, pw1 := io.Pipe()
	pr2, pw2 := io.Pipe()

	g.Go(func() error {
		defer func() {
			_ = pw1.CloseWithError(context.Cause(gCtx))
			_ = pw2.CloseWithError(context.Cause(gCtx))
		}()

		_, err := io.Copy(io.MultiWriter(pw1, pw2), r)
		if err != nil {
			return fmt.Errorf("item.getImageMetadata(%w): %v", apperr.ErrDebug, err)
		}
		return nil
	})

	g.Go(func() (err error) {
		defer func() {
			_ = pr1.CloseWithError(err)
		}()
		srcImg, _, err := image.Decode(pr1)
		if err != nil {
			return fmt.Errorf("item.getImageMetadata(%w): %v", apperr.ErrDebug, err)
		}

		bounds := srcImg.Bounds()
		width := bounds.Dx()
		height := bounds.Dy()

		if height > 100 || width > 100 {
			if width >= height {
				height = height / (width / 100)
				width = 100
			} else {
				width = width / (height / 100)
				height = 100
			}
		}
		dstRect := image.Rect(0, 0, width, height)
		dstImg := image.NewRGBA(dstRect)

		draw.BiLinear.Scale(dstImg, dstRect, srcImg, srcImg.Bounds(), draw.Over, nil)

		hash, err := blurhash.Encode(4, 3, dstImg)
		if err != nil {
			return fmt.Errorf("item.getImageMetadata(%w): %v", apperr.ErrDebug, err)
		}
		metadata = ImageMetadata{
			headerMetadata: header,
			Width:          bounds.Dx(),
			Height:         bounds.Dy(),
			BlurHash:       hash,
		}
		return nil
	})

	g.Go(func() (err error) {
		defer func() {
			_ = pr2.CloseWithError(err)
		}()
		config, _, err := image.DecodeConfig(pr2)
		if err != nil {
			return fmt.Errorf("item.getImageMetadata(%w): %v", apperr.ErrDebug, err)
		}
		if config.Width*config.Height*4 > 50<<20 {
			return apperr.ErrEntityTooLarge
		}
		_, _ = io.Copy(io.Discard, pr2)
		return nil
	})

	if err := g.Wait(); err != nil {
		return ImageMetadata{}, err
	}
	return metadata, nil
}

// getArchiveMetadata определяет тип архива и извлекает информацию о файлах (количество, разжатый размер, дерево)
func getArchiveMetadata(ctx context.Context, header headerMetadata, r io.Reader) (ArchiveMetadata, error) {
	// Identify авто-определяет формат архива/сжатия по имени файла и сигнатуре байт в r.
	// Возвращает streamReader со сброшенным/буферизованным потоком для последующего чтения.
	format, streamReader, err := archives.Identify(ctx, header.Filename, r)
	if err != nil {
		return ArchiveMetadata{}, fmt.Errorf("item.getArchiveMetadata(%w): %v", apperr.ErrDebug, err)
	}
	// Проверяем, поддерживается ли извлечение/обход файлов для данного формата
	extractor, ok := format.(archives.Extractor)
	if !ok {
		return ArchiveMetadata{}, fmt.Errorf("item.getArchiveMetadata(%w): format %T does not support extraction", apperr.ErrDebug, format)
	}
	var (
		fileCount             int
		uncompressedSizeBytes int64
		tree                  = make([]string, 0)
	)
	// Extract стримингово обходит все элементы архива без полной распаковки на диск
	err = extractor.Extract(ctx, streamReader, func(ctx context.Context, f archives.FileInfo) error {
		if !f.IsDir() {
			fileCount++
			uncompressedSizeBytes += f.Size()
		}
		name := f.NameInArchive
		if name == "" {
			name = f.Name()
		}
		// TODO: Добавить ограничение по количеству файлов
		tree = append(tree, name)
		return nil
	})
	if err != nil {
		return ArchiveMetadata{}, fmt.Errorf("item.getArchiveMetadata(%w): %v", apperr.ErrDebug, err)
	}
	return ArchiveMetadata{
		headerMetadata:        header,
		FileCount:             fileCount,
		UncompressedSizeBytes: uncompressedSizeBytes,
		Tree:                  tree,
	}, nil
}

// getModel3DMetadata декодирует GLTF/GLB модель и извлекает количество вершин, полигонов,
// наличие текстур, скелетного рига (rigging), анимаций и генератора рендера.
func getModel3DMetadata(ctx context.Context, header headerMetadata, r io.Reader) (Model3DMetadata, error) {
	doc := new(gltf.Document)
	decoder := gltf.NewDecoder(r)
	// NewDecoder умеет автоматически декодировать как .gltf (JSON), так и бинарный .glb поток
	if err := decoder.Decode(doc); err != nil {
		return Model3DMetadata{}, fmt.Errorf("item.getModel3DMetadata(%w): %v", apperr.ErrDebug, err)
	}
	var polygonCount int
	var vertexCount int
	// Обходим все меши и их примитивы для подсчета вершин и полигонов (треугольников)
	for _, mesh := range doc.Meshes {
		for _, primitive := range mesh.Primitives {
			// VertexCount: считываем количество вершин из accessor с атрибутом "POSITION"
			posIdx, hasPos := primitive.Attributes[gltf.POSITION]
			if hasPos && posIdx < len(doc.Accessors) {
				vertexCount += doc.Accessors[posIdx].Count
			}
			// PolygonCount: подсчитываем количество полигонов/треугольников
			if primitive.Indices != nil {
				idx := *primitive.Indices
				if idx < len(doc.Accessors) {
					count := doc.Accessors[idx].Count
					switch primitive.Mode {
					case gltf.PrimitiveTriangles: // Режим по умолчанию (3 индекса = 1 треугольник)
						polygonCount += count / 3
					case gltf.PrimitiveTriangleStrip, gltf.PrimitiveTriangleFan:
						if count >= 3 {
							polygonCount += count - 2
						}
					}
				}
			} else if hasPos && posIdx < len(doc.Accessors) {
				// Если нет индексов (non-indexed geometry)
				count := doc.Accessors[posIdx].Count
				switch primitive.Mode {
				case gltf.PrimitiveTriangles:
					polygonCount += count / 3
				case gltf.PrimitiveTriangleStrip, gltf.PrimitiveTriangleFan:
					if count >= 3 {
						polygonCount += count - 2
					}
				}
			}
		}
	}
	hasTexture := len(doc.Textures) > 0 || len(doc.Images) > 0
	hasRigging := len(doc.Skins) > 0
	hasAnimations := len(doc.Animations) > 0
	renderEngine := "PBR"
	if doc.Asset.Generator != "" {
		renderEngine = fmt.Sprintf("PBR (%s)", doc.Asset.Generator)
	}
	return Model3DMetadata{
		headerMetadata: header,
		PolygonCount:   polygonCount,
		VertexCount:    vertexCount,
		HasTexture:     hasTexture,
		HasRigging:     hasRigging,
		HasAnimations:  hasAnimations,
		RenderEngine:   renderEngine,
	}, nil
}

// appProperties структура для разбора метаданных docProps/app.xml в DOCX/PPTX
type appProperties struct {
	XMLName xml.Name `xml:"Properties"`
	Pages   int      `xml:"Pages"`
	Slides  int      `xml:"Slides"`
}

// getOfficePageCount считывает docProps/app.xml из ZIP-архива DOCX/PPTX и извлекает количество страниц или слайдов
func getOfficePageCount(r io.ReaderAt, size int64) (int, error) {
	zr, err := zip.NewReader(r, size)
	if err != nil {
		return 0, err
	}
	for _, f := range zr.File {
		if f.Name == "docProps/app.xml" {
			rc, err := f.Open()
			if err != nil {
				return 0, err
			}
			defer rc.Close()
			var props appProperties
			if err := xml.NewDecoder(rc).Decode(&props); err != nil {
				return 0, err
			}
			if props.Pages > 0 {
				return props.Pages, nil
			}
			if props.Slides > 0 {
				return props.Slides, nil
			}
			return 1, nil
		}
	}
	return 1, nil
}

// getDocumentMetadata подсчитывает количество страниц для PDF, DOCX, PPTX и других документов
func getDocumentMetadata(ctx context.Context, header headerMetadata, mType string, r io.Reader) (DocumentMetadata, error) {
	//TODO: ERROR Убрать потом ReadAll всё потоково
	data, err := io.ReadAll(r)
	if err != nil {
		return DocumentMetadata{}, fmt.Errorf("item.getDocumentMetadata(%w): %v", apperr.ErrDebug, err)
	}
	rs := bytes.NewReader(data)
	ext := strings.ToLower(filepath.Ext(header.Filename))
	pageCount := 1
	switch {
	case mType == "application/pdf" || ext == ".pdf":
		// pdfcpu считывает физическое количество страниц PDF
		count, err := api.PageCount(ctx, rs, nil)
		if err != nil {
			return DocumentMetadata{}, fmt.Errorf("item.getDocumentMetadata(%w): %v", apperr.ErrDebug, err)
		}
		pageCount = count
	case mType == "application/vnd.openxmlformats-officedocument.wordprocessingml.document" ||
		mType == "application/vnd.openxmlformats-officedocument.presentationml.presentation" ||
		ext == ".docx" || ext == ".pptx":
		// Разбор XML свойства docProps/app.xml
		count, err := getOfficePageCount(rs, int64(len(data)))
		if err == nil && count > 0 {
			pageCount = count
		}
	default:
		// Для обычных текстовых файлов (.txt, .csv, .md) по умолчанию 1 страница
		pageCount = 1
	}
	return DocumentMetadata{
		headerMetadata: header,
		PageCount:      pageCount,
	}, nil
}

// getAudioMetadata считывает аудиопоток и формат с помощью ffprobe
func getAudioMetadata(ctx context.Context, header headerMetadata, r io.Reader) (AudioMetadata, error) {
	data, err := ffprobe.ProbeReader(ctx, r)
	if err != nil {
		return AudioMetadata{}, fmt.Errorf("item.getAudioMetadata(%w): %v", apperr.ErrDebug, err)
	}
	var (
		duration     time.Duration
		sampleRateHZ int
		bitrateKbps  int
		channels     int
	)
	if data.Format != nil {
		duration = data.Format.Duration()
		if br, err := strconv.Atoi(data.Format.BitRate); err == nil && br > 0 {
			bitrateKbps = br / 1000
		}
	}
	audioStream := data.FirstAudioStream()
	if audioStream != nil {
		if sr, err := strconv.Atoi(audioStream.SampleRate); err == nil {
			sampleRateHZ = sr
		}
		if audioStream.Channels > 0 {
			channels = audioStream.Channels
		}
		if br, err := strconv.Atoi(audioStream.BitRate); err == nil && br > 0 {
			bitrateKbps = br / 1000
		}
	}
	return AudioMetadata{
		headerMetadata: header,
		Duration:       duration,
		SampleRateHZ:   sampleRateHZ,
		BitrateKbps:    bitrateKbps,
		Channels:       channels,
		WaveformData:   nil,
	}, nil
}

// getVideoMetadata считывает видеопоток (длительность, ширина, высота) с помощью ffprobe
func getVideoMetadata(ctx context.Context, header headerMetadata, r io.Reader) (VideoMetadata, error) {
	data, err := ffprobe.ProbeReader(ctx, r)
	if err != nil {
		return VideoMetadata{}, fmt.Errorf("item.getVideoMetadata(%w): %v", apperr.ErrDebug, err)
	}
	var (
		duration time.Duration
		width    int
		height   int
	)
	if data.Format != nil {
		duration = data.Format.Duration()
	}
	videoStream := data.FirstVideoStream()
	if videoStream != nil {
		width = videoStream.Width
		height = videoStream.Height
	}
	return VideoMetadata{
		headerMetadata: header,
		Duration:       duration,
		Width:          width,
		Height:         height,
	}, nil
}
