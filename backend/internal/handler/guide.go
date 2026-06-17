package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geojson"

	"github.com/colinbradley/sluff/internal/middleware"
	"github.com/colinbradley/sluff/internal/model"
	"github.com/colinbradley/sluff/internal/store"
)

// guideIDFromCtx extracts the authenticated guide ID from the request context.
func guideIDFromCtx(r *http.Request) string {
	return middleware.GuideIDFromContext(r.Context())
}

// GuideHandler implements the authenticated map and round CRUD endpoints.
type GuideHandler struct {
	store *store.SQLiteStore
}

// NewGuideHandler constructs a GuideHandler backed by the given store.
func NewGuideHandler(s *store.SQLiteStore) *GuideHandler {
	return &GuideHandler{store: s}
}

// CreateMap creates a new map owned by the authenticated guide.
func (h *GuideHandler) CreateMap(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	now := time.Now()
	m := &model.GameMap{
		ID:          uuid.New().String(),
		Name:        req.Name,
		Description: req.Description,
		GuideID:     guideIDFromCtx(r),
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := h.store.CreateMap(m); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create map")
		return
	}

	writeJSON(w, http.StatusCreated, m)
}

// ListMaps returns the maps owned by the authenticated guide.
func (h *GuideHandler) ListMaps(w http.ResponseWriter, r *http.Request) {
	guideID := guideIDFromCtx(r)
	maps, err := h.store.ListMapsByGuide(guideID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list maps")
		return
	}
	if maps == nil {
		maps = []model.GameMap{}
	}
	writeJSON(w, http.StatusOK, maps)
}

// GetMap returns the requested map (rounds included) if it is owned by the
// authenticated guide.
func (h *GuideHandler) GetMap(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "mapID")
	m, err := h.store.GetMap(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get map")
		return
	}
	if m == nil {
		writeError(w, http.StatusNotFound, "map not found")
		return
	}
	if m.GuideID != guideIDFromCtx(r) {
		writeError(w, http.StatusForbidden, "not your map")
		return
	}

	writeJSON(w, http.StatusOK, m)
}

// UpdateMap renames or re-describes a map owned by the authenticated guide.
func (h *GuideHandler) UpdateMap(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "mapID")

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	m, err := h.store.GetMap(id)
	if err != nil || m == nil {
		writeError(w, http.StatusNotFound, "map not found")
		return
	}
	if m.GuideID != guideIDFromCtx(r) {
		writeError(w, http.StatusForbidden, "not your map")
		return
	}

	if req.Name != "" {
		m.Name = req.Name
	}
	m.Description = req.Description

	if err := h.store.UpdateMap(m); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update map")
		return
	}

	writeJSON(w, http.StatusOK, m)
}

// DeleteMap deletes a map (and its rounds, via cascade) owned by the guide.
func (h *GuideHandler) DeleteMap(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "mapID")
	m, err := h.store.GetMap(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get map")
		return
	}
	if m == nil {
		writeError(w, http.StatusNotFound, "map not found")
		return
	}
	if m.GuideID != guideIDFromCtx(r) {
		writeError(w, http.StatusForbidden, "not your map")
		return
	}
	if err := h.store.DeleteMap(id); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete map")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type createRoundRequest struct {
	RoundNumber int             `json:"round_number"`
	Name        string          `json:"name"`
	StartPoint  json.RawMessage `json:"start_point"`
	EndPoint    json.RawMessage `json:"end_point"`
	Corridor    json.RawMessage `json:"corridor"`
	NoGoZones   json.RawMessage `json:"no_go_zones"`
}

// CreateRound adds a new round to a map owned by the authenticated guide.
func (h *GuideHandler) CreateRound(w http.ResponseWriter, r *http.Request) {
	mapID := chi.URLParam(r, "mapID")

	m, err := h.store.GetMap(mapID)
	if err != nil || m == nil {
		writeError(w, http.StatusNotFound, "map not found")
		return
	}
	if m.GuideID != guideIDFromCtx(r) {
		writeError(w, http.StatusForbidden, "not your map")
		return
	}

	var req createRoundRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	startPoint, err := parsePoint(req.StartPoint)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid start_point: "+err.Error())
		return
	}
	endPoint, err := parsePoint(req.EndPoint)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid end_point: "+err.Error())
		return
	}
	corridor, err := parsePolygon(req.Corridor)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid corridor: "+err.Error())
		return
	}

	round := &model.Round{
		ID:          uuid.New().String(),
		MapID:       mapID,
		RoundNumber: req.RoundNumber,
		Name:        req.Name,
		StartPoint:  startPoint,
		EndPoint:    endPoint,
		Corridor:    corridor,
	}

	if req.NoGoZones != nil {
		zones, err := parseNoGoZones(req.NoGoZones)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid no_go_zones: "+err.Error())
			return
		}
		round.NoGoZones = zones
	}

	if err := model.ValidateRound(round); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.store.CreateRound(round); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create round")
		return
	}

	// Return GeoJSON format for frontend consistency
	writeJSON(w, http.StatusCreated, round)
}

// UpdateRound modifies an existing round; only fields present in the request
// body are changed.
func (h *GuideHandler) UpdateRound(w http.ResponseWriter, r *http.Request) {
	roundID := chi.URLParam(r, "roundID")

	existing, err := h.store.GetRound(roundID)
	if err != nil || existing == nil {
		writeError(w, http.StatusNotFound, "round not found")
		return
	}
	parentMap, err := h.store.GetMap(existing.MapID)
	if err != nil || parentMap == nil {
		writeError(w, http.StatusInternalServerError, "failed to get map")
		return
	}
	if parentMap.GuideID != guideIDFromCtx(r) {
		writeError(w, http.StatusForbidden, "not your map")
		return
	}

	var req struct {
		RoundNumber *int            `json:"round_number"`
		Name        *string         `json:"name"`
		StartPoint  json.RawMessage `json:"start_point"`
		EndPoint    json.RawMessage `json:"end_point"`
		Corridor    json.RawMessage `json:"corridor"`
		NoGoZones   json.RawMessage `json:"no_go_zones"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.RoundNumber != nil {
		existing.RoundNumber = *req.RoundNumber
	}
	if req.Name != nil {
		existing.Name = *req.Name
	}
	if req.StartPoint != nil {
		sp, err := parsePoint(req.StartPoint)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid start_point: "+err.Error())
			return
		}
		existing.StartPoint = sp
	}
	if req.EndPoint != nil {
		ep, err := parsePoint(req.EndPoint)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid end_point: "+err.Error())
			return
		}
		existing.EndPoint = ep
	}
	if req.Corridor != nil {
		corridor, err := parsePolygon(req.Corridor)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid corridor: "+err.Error())
			return
		}
		existing.Corridor = corridor
	}

	if req.NoGoZones != nil {
		zones, err := parseNoGoZones(req.NoGoZones)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid no_go_zones: "+err.Error())
			return
		}
		existing.NoGoZones = zones
	}

	if err := model.ValidateRound(existing); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.store.UpdateRound(existing); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update round")
		return
	}

	// Return GeoJSON format for frontend consistency
	writeJSON(w, http.StatusOK, existing)
}

// DeleteRound removes a round from a map owned by the authenticated guide.
func (h *GuideHandler) DeleteRound(w http.ResponseWriter, r *http.Request) {
	roundID := chi.URLParam(r, "roundID")
	existing, err := h.store.GetRound(roundID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get round")
		return
	}
	if existing == nil {
		writeError(w, http.StatusNotFound, "round not found")
		return
	}
	parentMap, err := h.store.GetMap(existing.MapID)
	if err != nil || parentMap == nil {
		writeError(w, http.StatusInternalServerError, "failed to get map")
		return
	}
	if parentMap.GuideID != guideIDFromCtx(r) {
		writeError(w, http.StatusForbidden, "not your map")
		return
	}
	if err := h.store.DeleteRound(roundID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete round")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func parsePoint(raw json.RawMessage) (orb.Point, error) {
	geom, err := geojson.UnmarshalGeometry(raw)
	if err != nil {
		return orb.Point{}, err
	}
	p, ok := geom.Geometry().(orb.Point)
	if !ok {
		return orb.Point{}, errors.New("expected Point geometry")
	}
	return p, nil
}

func parsePolygon(raw json.RawMessage) (orb.Polygon, error) {
	geom, err := geojson.UnmarshalGeometry(raw)
	if err != nil {
		return nil, err
	}
	p, ok := geom.Geometry().(orb.Polygon)
	if !ok {
		return nil, errors.New("expected Polygon geometry")
	}
	return p, nil
}

// parseNoGoZones parses a JSON array of GeoJSON Polygon geometries.
func parseNoGoZones(raw json.RawMessage) ([]orb.Polygon, error) {
	var raws []json.RawMessage
	if err := json.Unmarshal(raw, &raws); err != nil {
		return nil, err
	}
	zones := make([]orb.Polygon, 0, len(raws))
	for _, r := range raws {
		poly, err := parsePolygon(r)
		if err != nil {
			return nil, err
		}
		zones = append(zones, poly)
	}
	return zones, nil
}
