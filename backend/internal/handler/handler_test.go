package handler_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/colinbradley/sluff/internal/handler"
	"github.com/colinbradley/sluff/internal/middleware"
	"github.com/colinbradley/sluff/internal/model"
	"github.com/colinbradley/sluff/internal/store"
	"github.com/colinbradley/sluff/internal/ws"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// testEnv bundles the dependencies every test needs.
type testEnv struct {
	store  *store.SQLiteStore
	hub    *ws.Hub
	guideH *handler.GuideHandler
	sessH  *handler.SessionHandler
	gameH  *handler.GameHandler
	router *chi.Mux
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()

	t.Setenv("ALLOW_DEFAULT_GUIDE", "true")
	s, err := store.NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create test store: %v", err)
	}
	t.Cleanup(func() { s.Close() })

	hub := ws.NewHub()
	go hub.Run()

	guideH := handler.NewGuideHandler(s)
	sessH := handler.NewSessionHandler(s)
	gameH := handler.NewGameHandler(s, hub)

	// Look up the seeded admin guide to inject into authenticated routes.
	adminGuide, err := s.GetGuideByUsername("admin")
	if err != nil || adminGuide == nil {
		t.Fatalf("failed to get seeded admin guide: %v", err)
	}
	injectGuide := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := middleware.WithGuideID(r.Context(), adminGuide.ID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}

	r := chi.NewRouter()

	// Guide routes (require guide ID in context in production)
	r.Route("/api/guide/maps", func(r chi.Router) {
		r.Use(injectGuide)
		r.Post("/", guideH.CreateMap)
		r.Get("/", guideH.ListMaps)
		r.Get("/{mapID}", guideH.GetMap)
		r.Put("/{mapID}", guideH.UpdateMap)
		r.Delete("/{mapID}", guideH.DeleteMap)
		r.Post("/{mapID}/rounds", guideH.CreateRound)
		r.Put("/{mapID}/rounds/{roundID}", guideH.UpdateRound)
		r.Delete("/{mapID}/rounds/{roundID}", guideH.DeleteRound)
	})

	// Session routes
	r.Route("/api/sessions", func(r chi.Router) {
		r.With(injectGuide).Post("/", sessH.CreateSession)
		r.With(injectGuide).Post("/solo", sessH.CreateSoloSession)
		r.Get("/{sessionID}", sessH.GetSession)
		r.Get("/code/{code}", sessH.GetSessionByCode)
		r.Post("/{sessionID}/join", sessH.JoinSession)
		r.Post("/{sessionID}/teams", sessH.CreateTeam)
		r.Post("/{sessionID}/teams/{teamID}/join", sessH.JoinTeam)
		r.Post("/{sessionID}/start", gameH.StartGame)
		r.Post("/{sessionID}/rounds/{roundID}/submit", gameH.SubmitRoute)
		r.Get("/{sessionID}/rounds/{roundID}/scores", gameH.GetScores)
	})

	return &testEnv{
		store:  s,
		hub:    hub,
		guideH: guideH,
		sessH:  sessH,
		gameH:  gameH,
		router: r,
	}
}

// doRequest executes a request against the test router and returns the
// recorder so callers can inspect status and body.
func (e *testEnv) doRequest(t *testing.T, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()

	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal request body: %v", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	req := httptest.NewRequest(method, path, bodyReader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	rr := httptest.NewRecorder()
	e.router.ServeHTTP(rr, req)
	return rr
}

// decodeJSON is a convenience to unmarshal the response body into dest.
func decodeJSON(t *testing.T, rr *httptest.ResponseRecorder, dest any) {
	t.Helper()
	if err := json.NewDecoder(rr.Body).Decode(dest); err != nil {
		t.Fatalf("decode response body: %v (body was: %s)", err, rr.Body.String())
	}
}

// jsonField extracts a top-level string field from an arbitrary JSON object.
func jsonField(t *testing.T, rr *httptest.ResponseRecorder, key string) string {
	t.Helper()
	var m map[string]any
	body := rr.Body.Bytes()
	if err := json.Unmarshal(body, &m); err != nil {
		t.Fatalf("unmarshal JSON for field %q: %v (body: %s)", key, err, string(body))
	}
	v, ok := m[key]
	if !ok {
		t.Fatalf("key %q not found in response: %s", key, string(body))
	}
	switch val := v.(type) {
	case string:
		return val
	default:
		t.Fatalf("key %q is not a string: %v", key, v)
		return ""
	}
}

// ---------------------------------------------------------------------------
// Sample GeoJSON constants used across tests
// ---------------------------------------------------------------------------

// Two distinct points roughly in the Wasatch mountains.
const (
	startPointJSON = `{"type":"Point","coordinates":[-111.58,40.59]}`
	endPointJSON   = `{"type":"Point","coordinates":[-111.56,40.61]}`
)

// A simple corridor polygon that encloses both points.
const corridorJSON = `{"type":"Polygon","coordinates":[[[-111.60,40.58],[-111.54,40.58],[-111.54,40.62],[-111.60,40.62],[-111.60,40.58]]]}`

// A LineString route that goes from start to end, staying inside the corridor.
const routeLineJSON = `{"type":"LineString","coordinates":[[-111.58,40.59],[-111.57,40.60],[-111.56,40.61]]}`

// buildRouterForGuide creates a minimal chi router that injects a specific
// guide ID for the guide-map endpoints, used by IDOR tests.
func buildRouterForGuide(s *store.SQLiteStore, guideID string) *chi.Mux {
	guideH := handler.NewGuideHandler(s)
	inject := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := middleware.WithGuideID(r.Context(), guideID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
	r := chi.NewRouter()
	r.Route("/api/guide/maps", func(r chi.Router) {
		r.Use(inject)
		r.Post("/", guideH.CreateMap)
		r.Get("/", guideH.ListMaps)
		r.Get("/{mapID}", guideH.GetMap)
		r.Put("/{mapID}", guideH.UpdateMap)
		r.Delete("/{mapID}", guideH.DeleteMap)
		r.Post("/{mapID}/rounds", guideH.CreateRound)
		r.Put("/{mapID}/rounds/{roundID}", guideH.UpdateRound)
		r.Delete("/{mapID}/rounds/{roundID}", guideH.DeleteRound)
	})
	return r
}

// ---------------------------------------------------------------------------
// IDOR: cross-guide ownership checks
// ---------------------------------------------------------------------------

func TestGuideIDOR(t *testing.T) {
	env := newTestEnv(t)

	// Register a second guide directly via the store.
	guide2 := &model.Guide{
		ID:           "guide2-id",
		Username:     "guide2",
		PasswordHash: "$2a$10$placeholder",
		CreatedAt:    time.Now(),
	}
	if err := env.store.CreateGuide(guide2); err != nil {
		t.Fatalf("create guide2: %v", err)
	}
	router2 := buildRouterForGuide(env.store, guide2.ID)

	// Create a map under the admin guide (guide1).
	rr := env.doRequest(t, http.MethodPost, "/api/guide/maps/", map[string]string{"name": "Guide1 Map"})
	if rr.Code != http.StatusCreated {
		t.Fatalf("create map: %d %s", rr.Code, rr.Body.String())
	}
	mapID := jsonField(t, rr, "id")

	// Create a round under guide1's map.
	rr = env.doRequest(t, http.MethodPost, "/api/guide/maps/"+mapID+"/rounds", map[string]any{
		"round_number": 1, "name": "R1",
		"start_point": json.RawMessage(startPointJSON),
		"end_point":   json.RawMessage(endPointJSON),
		"corridor":    json.RawMessage(corridorJSON),
	})
	if rr.Code != http.StatusCreated {
		t.Fatalf("create round: %d %s", rr.Code, rr.Body.String())
	}
	roundID := jsonField(t, rr, "id")

	doAs2 := func(t *testing.T, method, path string, body any) *httptest.ResponseRecorder {
		t.Helper()
		var bodyReader io.Reader
		if body != nil {
			b, _ := json.Marshal(body)
			bodyReader = bytes.NewReader(b)
		}
		req := httptest.NewRequest(method, path, bodyReader)
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		rr := httptest.NewRecorder()
		router2.ServeHTTP(rr, req)
		return rr
	}

	t.Run("GetMap_OtherGuide_403", func(t *testing.T) {
		rr := doAs2(t, http.MethodGet, "/api/guide/maps/"+mapID, nil)
		if rr.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("UpdateMap_OtherGuide_403", func(t *testing.T) {
		rr := doAs2(t, http.MethodPut, "/api/guide/maps/"+mapID, map[string]string{"name": "Hijacked"})
		if rr.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("DeleteMap_OtherGuide_403", func(t *testing.T) {
		rr := doAs2(t, http.MethodDelete, "/api/guide/maps/"+mapID, nil)
		if rr.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("CreateRound_OtherGuide_403", func(t *testing.T) {
		rr := doAs2(t, http.MethodPost, "/api/guide/maps/"+mapID+"/rounds", map[string]any{
			"round_number": 2, "name": "Evil Round",
			"start_point": json.RawMessage(startPointJSON),
			"end_point":   json.RawMessage(endPointJSON),
			"corridor":    json.RawMessage(corridorJSON),
		})
		if rr.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("UpdateRound_OtherGuide_403", func(t *testing.T) {
		name := "Hijacked Round"
		rr := doAs2(t, http.MethodPut, "/api/guide/maps/"+mapID+"/rounds/"+roundID, map[string]any{
			"name": &name,
		})
		if rr.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("DeleteRound_OtherGuide_403", func(t *testing.T) {
		rr := doAs2(t, http.MethodDelete, "/api/guide/maps/"+mapID+"/rounds/"+roundID, nil)
		if rr.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d: %s", rr.Code, rr.Body.String())
		}
	})
}

// ---------------------------------------------------------------------------
// Guide API Tests
// ---------------------------------------------------------------------------

func TestGuideAPI(t *testing.T) {
	env := newTestEnv(t)

	var mapID string

	t.Run("CreateMap_201", func(t *testing.T) {
		rr := env.doRequest(t, http.MethodPost, "/api/guide/maps/", map[string]string{
			"name":        "Wasatch Test Map",
			"description": "Integration test map",
		})
		if rr.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
		}
		var m map[string]any
		decodeJSON(t, rr, &m)
		if m["id"] == nil || m["id"] == "" {
			t.Fatal("response missing id")
		}
		if m["name"] != "Wasatch Test Map" {
			t.Fatalf("expected name 'Wasatch Test Map', got %v", m["name"])
		}
		mapID = m["id"].(string)
	})

	t.Run("ListMaps_200", func(t *testing.T) {
		rr := env.doRequest(t, http.MethodGet, "/api/guide/maps/", nil)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
		var maps []map[string]any
		decodeJSON(t, rr, &maps)
		if len(maps) < 1 {
			t.Fatal("expected at least 1 map")
		}
	})

	var roundID string

	t.Run("CreateRound_ValidGeoJSON_201", func(t *testing.T) {
		body := map[string]any{
			"round_number": 1,
			"name":         "Round 1",
			"start_point":  json.RawMessage(startPointJSON),
			"end_point":    json.RawMessage(endPointJSON),
			"corridor":     json.RawMessage(corridorJSON),
		}
		rr := env.doRequest(t, http.MethodPost, "/api/guide/maps/"+mapID+"/rounds", body)
		if rr.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
		}
		roundID = jsonField(t, rr, "id")
		if roundID == "" {
			t.Fatal("round id should not be empty")
		}
	})

	t.Run("GetMap_200_WithRounds", func(t *testing.T) {
		rr := env.doRequest(t, http.MethodGet, "/api/guide/maps/"+mapID, nil)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
		var m map[string]any
		decodeJSON(t, rr, &m)
		rounds, ok := m["rounds"].([]any)
		if !ok || len(rounds) != 1 {
			t.Fatalf("expected 1 round, got %v", m["rounds"])
		}
		r0 := rounds[0].(map[string]any)
		if r0["id"] != roundID {
			t.Fatalf("expected round id %s, got %v", roundID, r0["id"])
		}
		// Verify GeoJSON geometry on start_point
		sp := r0["start_point"].(map[string]any)
		if sp["type"] != "Point" {
			t.Fatalf("expected start_point type Point, got %v", sp["type"])
		}
	})

	t.Run("CreateRound_StartEqualsEnd_400", func(t *testing.T) {
		samePoint := `{"type":"Point","coordinates":[-111.58,40.59]}`
		body := map[string]any{
			"round_number": 2,
			"name":         "Bad Round",
			"start_point":  json.RawMessage(samePoint),
			"end_point":    json.RawMessage(samePoint),
			"corridor":     json.RawMessage(corridorJSON),
		}
		rr := env.doRequest(t, http.MethodPost, "/api/guide/maps/"+mapID+"/rounds", body)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
		}
		errMsg := jsonField(t, rr, "error")
		if errMsg == "" {
			t.Fatal("expected an error message")
		}
	})

	t.Run("CreateRound_MissingGeometry_400", func(t *testing.T) {
		body := map[string]any{
			"round_number": 3,
			"name":         "Incomplete Round",
			// Missing start_point, end_point, corridor
		}
		rr := env.doRequest(t, http.MethodPost, "/api/guide/maps/"+mapID+"/rounds", body)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("DeleteRound_204_ThenMapHasNoRounds", func(t *testing.T) {
		rr := env.doRequest(t, http.MethodDelete, "/api/guide/maps/"+mapID+"/rounds/"+roundID, nil)
		if rr.Code != http.StatusNoContent {
			t.Fatalf("expected 204, got %d: %s", rr.Code, rr.Body.String())
		}

		// Verify the map now has no rounds
		rr2 := env.doRequest(t, http.MethodGet, "/api/guide/maps/"+mapID, nil)
		if rr2.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr2.Code)
		}
		var m map[string]any
		decodeJSON(t, rr2, &m)
		rounds, _ := m["rounds"].([]any)
		if len(rounds) != 0 {
			t.Fatalf("expected 0 rounds after delete, got %d", len(rounds))
		}
	})
}

// ---------------------------------------------------------------------------
// Game Flow Smoke Test
// ---------------------------------------------------------------------------

func TestGameFlowSmoke(t *testing.T) {
	env := newTestEnv(t)

	// Step 1: Create a map
	rr := env.doRequest(t, http.MethodPost, "/api/guide/maps/", map[string]string{
		"name": "Smoke Test Map",
	})
	if rr.Code != http.StatusCreated {
		t.Fatalf("create map: expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	mapID := jsonField(t, rr, "id")

	// Step 2: Create a round
	roundBody := map[string]any{
		"round_number": 1,
		"name":         "Round 1",
		"start_point":  json.RawMessage(startPointJSON),
		"end_point":    json.RawMessage(endPointJSON),
		"corridor":     json.RawMessage(corridorJSON),
	}
	rr = env.doRequest(t, http.MethodPost, "/api/guide/maps/"+mapID+"/rounds", roundBody)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create round: expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	roundID := jsonField(t, rr, "id")

	// Step 3: Create a solo session
	rr = env.doRequest(t, http.MethodPost, "/api/sessions/solo", map[string]any{
		"map_id":         mapID,
		"player_name":    "test",
		"time_limit_sec": 300,
	})
	if rr.Code != http.StatusCreated {
		t.Fatalf("create solo session: expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	var soloResp struct {
		Session struct {
			ID           string `json:"id"`
			Phase        string `json:"phase"`
			CurrentRound int    `json:"current_round"`
		} `json:"session"`
		Player struct {
			ID string `json:"id"`
		} `json:"player"`
		Team struct {
			ID string `json:"id"`
		} `json:"team"`
	}
	decodeJSON(t, rr, &soloResp)

	sessionID := soloResp.Session.ID
	teamID := soloResp.Team.ID

	if sessionID == "" {
		t.Fatal("session id should not be empty")
	}
	if soloResp.Session.Phase != "waiting" {
		t.Fatalf("expected phase 'waiting', got %q", soloResp.Session.Phase)
	}

	// Step 4: Start the game
	t.Run("StartGame_SessionIsPlaying", func(t *testing.T) {
		rr := env.doRequest(t, http.MethodPost, "/api/sessions/"+sessionID+"/start", nil)
		if rr.Code != http.StatusOK {
			t.Fatalf("start game: expected 200, got %d: %s", rr.Code, rr.Body.String())
		}

		var sess struct {
			Phase        string `json:"phase"`
			CurrentRound int    `json:"current_round"`
		}
		decodeJSON(t, rr, &sess)
		if sess.Phase != "playing" {
			t.Fatalf("expected phase 'playing', got %q", sess.Phase)
		}
		if sess.CurrentRound != 1 {
			t.Fatalf("expected current_round 1, got %d", sess.CurrentRound)
		}
	})

	// Step 5: Submit a route and verify scoring
	t.Run("SubmitRoute_ScoredResponse", func(t *testing.T) {
		body := map[string]any{
			"team_id": teamID,
			"path":    json.RawMessage(routeLineJSON),
		}
		rr := env.doRequest(t, http.MethodPost, "/api/sessions/"+sessionID+"/rounds/"+roundID+"/submit", body)
		if rr.Code != http.StatusCreated {
			t.Fatalf("submit route: expected 201, got %d: %s", rr.Code, rr.Body.String())
		}

		var route struct {
			ID      string   `json:"id"`
			Score   *float64 `json:"score"`
			Details *struct {
				FinalScore        float64 `json:"final_score"`
				PercentInCorridor float64 `json:"percent_in_corridor"`
				ConnectsStart     bool    `json:"connects_start"`
				ConnectsEnd       bool    `json:"connects_end"`
			} `json:"details"`
		}
		decodeJSON(t, rr, &route)

		if route.ID == "" {
			t.Fatal("route id should not be empty")
		}
		if route.Score == nil {
			t.Fatal("score should not be nil")
		}
		if *route.Score <= 0 {
			t.Fatalf("score should be positive, got %f", *route.Score)
		}
		if route.Details == nil {
			t.Fatal("details should not be nil")
		}
		if route.Details.FinalScore <= 0 {
			t.Fatalf("final_score should be positive, got %f", route.Details.FinalScore)
		}
		// The route starts/ends exactly at start/end points, so endpoint
		// connection should succeed.
		if !route.Details.ConnectsStart {
			t.Error("expected connects_start to be true")
		}
		if !route.Details.ConnectsEnd {
			t.Error("expected connects_end to be true")
		}
	})
}
