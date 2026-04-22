package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geojson"

	"github.com/colinbradley/sluff/internal/model"
	"github.com/colinbradley/sluff/internal/store"
)

type GuideHandler struct {
	store store.Store
}

func NewGuideHandler(s store.Store) *GuideHandler {
	return &GuideHandler{store: s}
}

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
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := h.store.CreateMap(m); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create map")
		return
	}

	writeJSON(w, http.StatusCreated, m)
}

func (h *GuideHandler) ListMaps(w http.ResponseWriter, r *http.Request) {
	maps, err := h.store.ListMaps()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list maps")
		return
	}
	if maps == nil {
		maps = []model.GameMap{}
	}
	writeJSON(w, http.StatusOK, maps)
}

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

	// Convert orb geometries to GeoJSON format for frontend compatibility
	roundsJSON := make([]interface{}, len(m.Rounds))
	for i, round := range m.Rounds {
		roundsJSON[i] = roundToGeoJSON(&round)
	}

	mapJSON := map[string]interface{}{
		"id":          m.ID,
		"name":        m.Name,
		"description": m.Description,
		"created_at":  m.CreatedAt,
		"updated_at":  m.UpdatedAt,
		"rounds":      roundsJSON,
	}

	writeJSON(w, http.StatusOK, mapJSON)
}

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

func (h *GuideHandler) DeleteMap(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "mapID")
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

func (h *GuideHandler) CreateRound(w http.ResponseWriter, r *http.Request) {
	mapID := chi.URLParam(r, "mapID")

	m, err := h.store.GetMap(mapID)
	if err != nil || m == nil {
		writeError(w, http.StatusNotFound, "map not found")
		return
	}

	var req createRoundRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	startPoint, endPoint, corridor, err := model.RoundFromJSON(
		string(req.StartPoint), string(req.EndPoint), string(req.Corridor),
	)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid geometry: "+err.Error())
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
	writeJSON(w, http.StatusCreated, roundToGeoJSON(round))
}

func (h *GuideHandler) UpdateRound(w http.ResponseWriter, r *http.Request) {
	roundID := chi.URLParam(r, "roundID")

	existing, err := h.store.GetRound(roundID)
	if err != nil || existing == nil {
		writeError(w, http.StatusNotFound, "round not found")
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
		sp, _, _, err := model.RoundFromJSON(string(req.StartPoint), string(req.EndPoint), string(req.Corridor))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid geometry")
			return
		}
		existing.StartPoint = sp
	}
	if req.EndPoint != nil {
		existing.EndPoint, _, _, _ = model.RoundFromJSON(
			toGeoJSONPoint(existing.StartPoint), string(req.EndPoint), toGeoJSONPolygon(existing.Corridor),
		)
	}
	if req.Corridor != nil {
		_, _, corridor, err := model.RoundFromJSON(
			toGeoJSONPoint(existing.StartPoint), toGeoJSONPoint(existing.EndPoint), string(req.Corridor),
		)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid corridor geometry")
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
	writeJSON(w, http.StatusOK, roundToGeoJSON(existing))
}

func (h *GuideHandler) DeleteRound(w http.ResponseWriter, r *http.Request) {
	roundID := chi.URLParam(r, "roundID")
	if err := h.store.DeleteRound(roundID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete round")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// helpers

// roundToGeoJSON converts a Round with orb types to a map with GeoJSON geometry
func roundToGeoJSON(round *model.Round) map[string]interface{} {
	startGeo := map[string]interface{}{
		"type":        "Point",
		"coordinates": []float64{round.StartPoint[0], round.StartPoint[1]},
	}
	endGeo := map[string]interface{}{
		"type":        "Point",
		"coordinates": []float64{round.EndPoint[0], round.EndPoint[1]},
	}

	noGoGeo := make([]interface{}, len(round.NoGoZones))
	for i, zone := range round.NoGoZones {
		noGoGeo[i] = polygonToGeoJSON(zone)
	}

	return map[string]interface{}{
		"id":           round.ID,
		"map_id":       round.MapID,
		"round_number": round.RoundNumber,
		"name":         round.Name,
		"start_point":  startGeo,
		"end_point":    endGeo,
		"corridor":     polygonToGeoJSON(round.Corridor),
		"no_go_zones":  noGoGeo,
	}
}

func polygonToGeoJSON(p orb.Polygon) map[string]interface{} {
	coords := make([][][]float64, len(p))
	for i, ring := range p {
		coords[i] = make([][]float64, len(ring))
		for j, pt := range ring {
			coords[i][j] = []float64{pt[0], pt[1]}
		}
	}
	return map[string]interface{}{
		"type":        "Polygon",
		"coordinates": coords,
	}
}

// parseNoGoZones parses a JSON array of GeoJSON Polygon geometries.
func parseNoGoZones(raw json.RawMessage) ([]orb.Polygon, error) {
	var raws []json.RawMessage
	if err := json.Unmarshal(raw, &raws); err != nil {
		return nil, err
	}
	zones := make([]orb.Polygon, 0, len(raws))
	for _, r := range raws {
		geom, err := geojson.UnmarshalGeometry(r)
		if err != nil {
			return nil, err
		}
		poly, ok := geom.Geometry().(orb.Polygon)
		if !ok {
			return nil, fmt.Errorf("expected Polygon geometry")
		}
		zones = append(zones, poly)
	}
	return zones, nil
}

func toGeoJSONPoint(p orb.Point) string {
	b, _ := json.Marshal(map[string]interface{}{
		"type":        "Point",
		"coordinates": []float64{p[0], p[1]},
	})
	return string(b)
}

func toGeoJSONPolygon(p orb.Polygon) string {
	coords := make([][][]float64, len(p))
	for i, ring := range p {
		coords[i] = make([][]float64, len(ring))
		for j, pt := range ring {
			coords[i][j] = []float64{pt[0], pt[1]}
		}
	}
	b, _ := json.Marshal(map[string]interface{}{
		"type":        "Polygon",
		"coordinates": coords,
	})
	return string(b)
}
