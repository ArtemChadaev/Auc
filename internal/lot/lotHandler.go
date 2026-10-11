package lot

import (
	"fmt"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"uuid"

	"github.com/ArtemChadaev/Auction/cmd/apperr"
	"github.com/ArtemChadaev/Auction/cmd/httpx"
)

// Handler handles HTTP requests for lots.
type Handler struct {
	mu   sync.RWMutex
	lots []Lot
}

// NewHandler initializes a new lot handler populated with realistic mock data.
func NewHandler() *Handler {
	h := &Handler{}
	h.lots = initMockLots()
	return h
}

// Routes returns the ServeMux with lot routes registered.
func Routes(h *Handler) http.Handler {
	return h.Routes()
}

// Routes configures endpoints for the lot resource.
func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()

	// GET /{$} -> List lots with filtering and pagination
	mux.HandleFunc("GET /{$}", h.listLots)

	// GET /{id} -> Get lot by ID
	mux.HandleFunc("GET /{id}", h.getLotByID)

	return mux
}

// @Summary Получить список лотов
// @Description Возвращает список лотов аукциона с фильтрацией, сортировкой и пагинацией
// @Tags lot
// @Produce json
// @Param q query string false "Поисковый запрос"
// @Param search query string false "Алиас для поискового запроса"
// @Param category query string false "Категория лота"
// @Param status query string false "Статус лота (draft, pending_review, approved, active, rejected, finished, cancelled)"
// @Param sort query string false "Сортировка (price_asc, price_desc, ending_soon, popular, newest)"
// @Param min_price query int false "Минимальная цена"
// @Param max_price query int false "Максимальная цена"
// @Param page query int false "Номер страницы (по умолчанию 1)"
// @Param limit query int false "Лимит на страницу (по умолчанию 20, максимум 100)"
// @Success 200 {object} httpx.Response{data=LotListResponse}
// @Failure 500 {object} httpx.Response "Внутренняя ошибка сервера"
// @Router /api/lot/ [get]
func (h *Handler) listLots(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	filter := LotFilter{
		Query:    strings.ToLower(strings.TrimSpace(query.Get("q"))),
		Category: strings.ToLower(strings.TrimSpace(query.Get("category"))),
		Status:   strings.ToLower(strings.TrimSpace(query.Get("status"))),
		Sort:     strings.ToLower(strings.TrimSpace(query.Get("sort"))),
		Page:     1,
		Limit:    20,
	}

	// Also support ?search= as an alias for ?q=
	if filter.Query == "" {
		filter.Query = strings.ToLower(strings.TrimSpace(query.Get("search")))
	}

	if minPriceStr := query.Get("min_price"); minPriceStr != "" {
		if val, err := strconv.ParseInt(minPriceStr, 10, 64); err == nil && val >= 0 {
			filter.MinPrice = &val
		}
	}

	if maxPriceStr := query.Get("max_price"); maxPriceStr != "" {
		if val, err := strconv.ParseInt(maxPriceStr, 10, 64); err == nil && val >= 0 {
			filter.MaxPrice = &val
		}
	}

	if pageStr := query.Get("page"); pageStr != "" {
		if val, err := strconv.Atoi(pageStr); err == nil && val > 0 {
			filter.Page = val
		}
	}

	if limitStr := query.Get("limit"); limitStr != "" {
		if val, err := strconv.Atoi(limitStr); err == nil && val > 0 {
			if val > 100 {
				val = 100
			}
			filter.Limit = val
		}
	}

	h.mu.RLock()
	var filtered []Lot
	for _, item := range h.lots {
		// Filter by query (title, description text, short text, category)
		if filter.Query != "" {
			titleMatch := strings.Contains(strings.ToLower(item.Title), filter.Query)
			descMatch := strings.Contains(strings.ToLower(item.Description.Text), filter.Query)
			shortDescMatch := strings.Contains(strings.ToLower(item.Description.ShortText), filter.Query)
			catMatch := strings.Contains(strings.ToLower(item.Category), filter.Query)
			if !titleMatch && !descMatch && !shortDescMatch && !catMatch {
				continue
			}
		}

		// Filter by category
		if filter.Category != "" && strings.ToLower(item.Category) != filter.Category {
			continue
		}

		// Filter by status
		if filter.Status != "" && strings.ToLower(item.Status) != filter.Status {
			continue
		}

		// Filter by price range
		if filter.MinPrice != nil && item.CurrentBid < *filter.MinPrice {
			continue
		}
		if filter.MaxPrice != nil && item.CurrentBid > *filter.MaxPrice {
			continue
		}

		filtered = append(filtered, item)
	}
	h.mu.RUnlock()

	// Apply sorting
	switch filter.Sort {
	case "price_asc":
		sort.Slice(filtered, func(i, j int) bool {
			return filtered[i].CurrentBid < filtered[j].CurrentBid
		})
	case "price_desc":
		sort.Slice(filtered, func(i, j int) bool {
			return filtered[i].CurrentBid > filtered[j].CurrentBid
		})
	case "ending_soon":
		sort.Slice(filtered, func(i, j int) bool {
			if filtered[i].EndsAt == nil {
				return false
			}
			if filtered[j].EndsAt == nil {
				return true
			}
			return filtered[i].EndsAt.Before(*filtered[j].EndsAt)
		})
	case "popular":
		sort.Slice(filtered, func(i, j int) bool {
			return filtered[i].BidsCount > filtered[j].BidsCount
		})
	default: // "newest" or default
		sort.Slice(filtered, func(i, j int) bool {
			return filtered[i].CreatedAt.After(filtered[j].CreatedAt)
		})
	}

	total := len(filtered)
	totalPages := 0
	if total > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(filter.Limit)))
	}

	start := (filter.Page - 1) * filter.Limit
	var paged []Lot
	if start < total {
		end := start + filter.Limit
		if end > total {
			end = total
		}
		paged = filtered[start:end]
	} else {
		paged = []Lot{}
	}

	response := LotListResponse{
		Items:      paged,
		Total:      total,
		Page:       filter.Page,
		Limit:      filter.Limit,
		TotalPages: totalPages,
	}

	httpx.WriteResponse(w, httpx.Response{
		Code: http.StatusOK,
		Data: response,
	})
}

// @Summary Получить лот по ID
// @Description Возвращает детальную информацию о лоте аукциона
// @Tags lot
// @Produce json
// @Param id path int true "ID лота"
// @Success 200 {object} httpx.Response{data=Lot}
// @Failure 400 {object} httpx.Response "Неверный ID лота"
// @Failure 404 {object} httpx.Response "Лот не найден"
// @Router /api/lot/{id} [get]
func (h *Handler) getLotByID(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		appErr := fmt.Errorf("getLotByID.ParseInt: %w", ErrInvalidLotID)
		apperr.Log(r.Context(), "getLotByID.ParseInt", appErr)
		httpx.WriteResponse(w, httpx.ErrRespInvalidReqBody)
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, lot := range h.lots {
		if lot.ID == id {
			httpx.WriteResponse(w, httpx.Response{
				Code: http.StatusOK,
				Data: lot,
			})
			return
		}
	}

	httpx.WriteResponse(w, httpx.ErrRespNotFound)
}

func initMockLots() []Lot {
	now := time.Now().UTC()

	seller1ID := uuid.MustParse("0191ec4d-9a3b-7221-a53c-1b9e67d40001")
	seller1AvatarID := uuid.MustParse("0191ec4d-9a3b-7221-a53c-1b9e67d400a1")

	seller2ID := uuid.MustParse("0191ec4d-9a3b-7221-a53c-1b9e67d40002")
	seller2AvatarID := uuid.MustParse("0191ec4d-9a3b-7221-a53c-1b9e67d400a2")

	seller3ID := uuid.MustParse("0191ec4d-9a3b-7221-a53c-1b9e67d40003")
	seller3AvatarID := uuid.MustParse("0191ec4d-9a3b-7221-a53c-1b9e67d400a3")

	winnerID := uuid.MustParse("0191ec4d-9a3b-7221-a53c-1b9e67d40099")

	lot1Started := now.Add(-12 * time.Hour)
	lot1Ends := now.Add(4 * time.Hour)

	lot2Started := now.Add(-2 * time.Hour)
	lot2Ends := now.Add(22 * time.Hour)

	lot3Started := now.Add(-6 * time.Hour)
	lot3Ends := now.Add(8 * time.Hour)

	lot4Started := now.Add(-48 * time.Hour)
	lot4Ends := now.Add(-2 * time.Hour)
	lot4Finished := lot4Ends
	lot4FinishAmount := int64(2400)

	lot5Started := now.Add(-1 * time.Hour)
	lot5Ends := now.Add(18 * time.Hour)

	vidThumb1 := uuid.MustParse("0191ec4d-9a3b-7221-a53c-1b9e67d40103")
	vidThumb3 := uuid.MustParse("0191ec4d-9a3b-7221-a53c-1b9e67d40303")
	vidThumb5 := uuid.MustParse("0191ec4d-9a3b-7221-a53c-1b9e67d40503")

	return []Lot{
		{
			ID:       1,
			SellerID: seller1ID,
			Seller: SellerInfo{
				ID:       seller1ID,
				Name:     "CyberForge Studio",
				AvatarID: &seller1AvatarID,
				Rating:   4.95,
			},
			ItemID: 101,
			ItemSummary: &ItemSummary{
				ID:       101,
				Type:     "3d",
				Preview:  uuid.MustParse("0191ec4d-9a3b-7221-a53c-1b9e67d40101"),
				BlurHash: "LEHLk[WB2yk8pyoJadR*.7kCMdnj",
			},
			Status:   StatusActive,
			Category: "3d_models",
			Title:    "Cyber Samurai 3D Low-Poly Character Rigged",
			Description: LotDescription{
				Text:      "Fully rigged low-poly sci-fi character designed for next-gen game engines. Includes 4K PBR textures, modular armor parts, katana with emission shaders, and 12 custom animations (Walk, Run, Idle, Light Attack, Heavy Attack, Death). Compatible with Unreal Engine 5 Humanoid Skeleton and Unity Mecanim.",
				ShortText: "Game-ready low-poly cyberpunk samurai with 4K PBR textures and full humanoid rig.",
				Media: []MediaItem{
					{
						ID:       uuid.MustParse("0191ec4d-9a3b-7221-a53c-1b9e67d40101"),
						Type:     MediaTypeImage,
						BlurHash: "LEHLk[WB2yk8pyoJadR*.7kCMdnj",
						Width:    1920,
						Height:   1080,
						Order:    1,
						Caption:  "Beauty Render - Front 3/4 View",
					},
					{
						ID:       uuid.MustParse("0191ec4d-9a3b-7221-a53c-1b9e67d40102"),
						Type:     MediaTypeImage,
						BlurHash: "L6PZfSi_.AyE_3t7t7R**0o#DgR4",
						Width:    1920,
						Height:   1080,
						Order:    2,
						Caption:  "Wireframe & UV Topology",
					},
					{
						ID:          uuid.MustParse("0191ec4d-9a3b-7221-a53c-1b9e67d40104"),
						Type:        MediaTypeVideo,
						BlurHash:    "LKO:N_j[00f6~qj[fQj[00fQ~qf6",
						Width:       1920,
						Height:      1080,
						DurationSec: 38.5,
						ThumbnailID: &vidThumb1,
						Order:       3,
						Caption:     "360 Turntable & Combat Animations Demonstration",
					},
				},
				Attributes: map[string]string{
					"Poly Count":     "14,850 tris",
					"Vertices":       "8,210",
					"Textures":       "4K PBR (Albedo, Normal, Roughness, Metallic, Emissive)",
					"Rig":            "Full Humanoid IK/FK",
					"Formats":        "FBX, OBJ, BLEND",
					"Engine Support": "Unreal Engine 5.3+, Unity 2022.3+",
				},
			},
			StartAmount: 500,
			CurrentBid:  1450,
			MinStep:     50,
			BidsCount:   19,
			CreatedAt:   now.Add(-24 * time.Hour),
			StartedAt:   &lot1Started,
			EndsAt:      &lot1Ends,
		},
		{
			ID:       2,
			SellerID: seller2ID,
			Seller: SellerInfo{
				ID:       seller2ID,
				Name:     "VintageOptics Collector",
				AvatarID: &seller2AvatarID,
				Rating:   5.0,
			},
			ItemID: 102,
			ItemSummary: &ItemSummary{
				ID:       102,
				Type:     "image",
				Preview:  uuid.MustParse("0191ec4d-9a3b-7221-a53c-1b9e67d40201"),
				BlurHash: "LGF5?xYk^6#M@-5c,1J5@[5#slR*",
			},
			Status:   StatusActive,
			Category: "collectibles",
			Title:    "Leica M3 Single Stroke (1958) + Summicron 50mm f/2",
			Description: LotDescription{
				Text:      "Authentic 1958 Leica M3 Single Stroke rangefinder in near-mint collector condition (serial #918421). Complete with collapsible Leitz Summicron 50mm f/2 lens. Serviced in Germany in 2024: shutter speeds calibrated, rangefinder aligned, viewfinder bright with spotless contrast.",
				ShortText: "Iconic 1958 Leica M3 Single Stroke 35mm rangefinder with serviced Summicron 50mm f/2.",
				Media: []MediaItem{
					{
						ID:       uuid.MustParse("0191ec4d-9a3b-7221-a53c-1b9e67d40201"),
						Type:     MediaTypeImage,
						BlurHash: "LGF5?xYk^6#M@-5c,1J5@[5#slR*",
						Width:    2400,
						Height:   1600,
						Order:    1,
						Caption:  "Front View with Leitz Summicron 50mm f/2",
					},
					{
						ID:       uuid.MustParse("0191ec4d-9a3b-7221-a53c-1b9e67d40202"),
						Type:     MediaTypeImage,
						BlurHash: "L8G@1?%200M{~qof?bof00Rj?bWB",
						Width:    2400,
						Height:   1600,
						Order:    2,
						Caption:  "Top Plate Engravings & Serial Number",
					},
					{
						ID:       uuid.MustParse("0191ec4d-9a3b-7221-a53c-1b9e67d40203"),
						Type:     MediaTypeImage,
						BlurHash: "LIK2;b~q009F00?b_3%M00t7_3WB",
						Width:    2400,
						Height:   1600,
						Order:    3,
						Caption:  "Open Film Chamber & Pressure Plate Condition",
					},
				},
				Attributes: map[string]string{
					"Manufacturer":  "Ernst Leitz Wetzlar",
					"Year":          "1958",
					"Serial Number": "918421",
					"Condition":     "Collector Grade (9.5/10)",
					"Mount":         "Leica M Bayonet",
					"Lens":          "Leitz Summicron 50mm f/2 (Collapsible)",
				},
			},
			StartAmount: 1800,
			CurrentBid:  3200,
			MinStep:     100,
			BidsCount:   14,
			CreatedAt:   now.Add(-10 * time.Hour),
			StartedAt:   &lot2Started,
			EndsAt:      &lot2Ends,
		},
		{
			ID:       3,
			SellerID: seller3ID,
			Seller: SellerInfo{
				ID:       seller3ID,
				Name:     "Algorithmic Horizons",
				AvatarID: &seller3AvatarID,
				Rating:   4.88,
			},
			ItemID: 103,
			ItemSummary: &ItemSummary{
				ID:       103,
				Type:     "video",
				Preview:  uuid.MustParse("0191ec4d-9a3b-7221-a53c-1b9e67d40301"),
				BlurHash: "LUC?g04n%Mxt~qoft7of%MRjIUWB",
			},
			Status:   StatusActive,
			Category: "art",
			Title:    "Infinite Horizons #42 - Procedural Vector Canvas",
			Description: LotDescription{
				Text:      "1/1 Generative vector artwork dynamically constructed through high-dimensional noise flow fields. Includes master losslessly compressed SVG and TIFF (8K resolution), custom interactive GLSL shader bundle, and a signed 60x90cm physical Hahnemühle fine art print delivered worldwide.",
				ShortText: "1/1 Algorithmic generative art piece with 8K master vectors, code and signed physical print.",
				Media: []MediaItem{
					{
						ID:       uuid.MustParse("0191ec4d-9a3b-7221-a53c-1b9e67d40301"),
						Type:     MediaTypeImage,
						BlurHash: "LUC?g04n%Mxt~qoft7of%MRjIUWB",
						Width:    3840,
						Height:   2160,
						Order:    1,
						Caption:  "High-Resolution Master Rendering (8K Master)",
					},
					{
						ID:          uuid.MustParse("0191ec4d-9a3b-7221-a53c-1b9e67d40302"),
						Type:        MediaTypeVideo,
						BlurHash:    "L784:c%M00_300of_3xu00Rj?bof",
						Width:       1920,
						Height:      1080,
						DurationSec: 42.0,
						ThumbnailID: &vidThumb3,
						Order:       2,
						Caption:     "Realtime Flow Field Simulation Loop (60 FPS)",
					},
				},
				Attributes: map[string]string{
					"Medium":         "Generative GLSL Algorithm",
					"Edition":        "1 of 1",
					"Master Format":  "SVG, 8K TIFF (300 DPI)",
					"Physical Print": "Hahnemühle Photo Rag 308gsm, 60x90 cm",
					"Certificate":    "Digitally signed & cryptographic provenance",
				},
			},
			StartAmount: 300,
			CurrentBid:  780,
			MinStep:     30,
			BidsCount:   8,
			CreatedAt:   now.Add(-18 * time.Hour),
			StartedAt:   &lot3Started,
			EndsAt:      &lot3Ends,
		},
		{
			ID:       4,
			SellerID: seller1ID,
			Seller: SellerInfo{
				ID:       seller1ID,
				Name:     "KeebEnthusiast",
				AvatarID: &seller1AvatarID,
				Rating:   4.95,
			},
			ItemID: 104,
			ItemSummary: &ItemSummary{
				ID:       104,
				Type:     "image",
				Preview:  uuid.MustParse("0191ec4d-9a3b-7221-a53c-1b9e67d40401"),
				BlurHash: "LHD*yB_300M{00IU?bxu00t7_3of",
			},
			Status:   StatusFinished,
			Category: "electronics",
			Title:    "Custom Keycult No. 1/65 Brass Edition Mechanical Keyboard",
			Description: LotDescription{
				Text:      "Completed auction. Brand new pristine Keycult No. 1/65 with mirror-polished brass mid-piece and bottom weight. Built with lubed & filmed Gateron Oil Kings (Krytox 205g0) on polycarbonate half-plate. Keycaps: GMK Dualshot 2 Base + Accent kit.",
				ShortText: "Completed auction: Custom Keycult No. 1/65 mechanical keyboard with polished brass weight.",
				Media: []MediaItem{
					{
						ID:       uuid.MustParse("0191ec4d-9a3b-7221-a53c-1b9e67d40401"),
						Type:     MediaTypeImage,
						BlurHash: "LHD*yB_300M{00IU?bxu00t7_3of",
						Width:    2048,
						Height:   1365,
						Order:    1,
						Caption:  "Assembled Board with GMK Dualshot 2",
					},
					{
						ID:       uuid.MustParse("0191ec4d-9a3b-7221-a53c-1b9e67d40402"),
						Type:     MediaTypeImage,
						BlurHash: "LMC$Q_of00M{00j[?bof00of_3j[",
						Width:    2048,
						Height:   1365,
						Order:    2,
						Caption:  "Mirror-Polished Brass Bottom Weight",
					},
				},
				Attributes: map[string]string{
					"Case":       "Keycult No. 1/65",
					"Mid-piece":  "Mirror-Polished Brass",
					"Plate":      "Polycarbonate Half-Plate",
					"Switches":   "Gateron Oil Kings (Krytox 205g0 / Deskeys Films)",
					"Stabilizer": "Durock V2 Lubed (XHT-BDZ)",
					"Keycaps":    "GMK Dualshot 2",
				},
			},
			StartAmount:  1200,
			CurrentBid:   2400,
			MinStep:      100,
			FinishAmount: &lot4FinishAmount,
			BidsCount:    31,
			Winner: &WinnerInfo{
				ID:   winnerID,
				Name: "TypingMaster",
			},
			CreatedAt:  now.Add(-72 * time.Hour),
			StartedAt:  &lot4Started,
			EndsAt:     &lot4Ends,
			FinishedAt: &lot4Finished,
		},
		{
			ID:       5,
			SellerID: seller2ID,
			Seller: SellerInfo{
				ID:       seller2ID,
				Name:     "RetroWave Records",
				AvatarID: &seller2AvatarID,
				Rating:   4.92,
			},
			ItemID: 105,
			ItemSummary: &ItemSummary{
				ID:       105,
				Type:     "audio",
				Preview:  uuid.MustParse("0191ec4d-9a3b-7221-a53c-1b9e67d40501"),
				BlurHash: "LfG[mR-;_4%M~qoft7of%MRjIUof",
			},
			Status:   StatusActive,
			Category: "audio",
			Title:    "Synthwave Odyssey - 10 Full Audio Tracks + Stems",
			Description: LotDescription{
				Text:      "Exclusive full commercial buyout for 10 unreleased authentic analog synthwave tracks produced with Roland Juno-106, Moog Sub 37, and LinnDrum hardware. Includes 24-bit 48kHz WAV masters, complete multitrack stems, and MIDI compositions.",
				ShortText: "10 unreleased analog synthwave tracks with multitrack stems and full commercial rights.",
				Media: []MediaItem{
					{
						ID:       uuid.MustParse("0191ec4d-9a3b-7221-a53c-1b9e67d40501"),
						Type:     MediaTypeImage,
						BlurHash: "LfG[mR-;_4%M~qoft7of%MRjIUof",
						Width:    1400,
						Height:   1400,
						Order:    1,
						Caption:  "Album Artwork & Vinyl Sleeve Concept",
					},
					{
						ID:          uuid.MustParse("0191ec4d-9a3b-7221-a53c-1b9e67d40502"),
						Type:        MediaTypeVideo,
						BlurHash:    "LLBN6y4n00_300of_3xu00Rj?bof",
						Width:       1920,
						Height:      1080,
						DurationSec: 60.0,
						ThumbnailID: &vidThumb5,
						Order:       2,
						Caption:     "Hardware Studio Session & Audio Preview Reel",
					},
				},
				Attributes: map[string]string{
					"Track Count":    "10 Full Length Tracks",
					"Format":         "WAV 24-bit / 48kHz, FLAC, MIDI",
					"License":        "Full Exclusive Commercial Buyout",
					"Hardware Used":  "Roland Juno-106, Moog Sub 37, LinnDrum LM-2",
					"Total Duration": "39 min 14 sec",
				},
			},
			StartAmount: 250,
			CurrentBid:  600,
			MinStep:     25,
			BidsCount:   15,
			CreatedAt:   now.Add(-8 * time.Hour),
			StartedAt:   &lot5Started,
			EndsAt:      &lot5Ends,
		},
	}
}
