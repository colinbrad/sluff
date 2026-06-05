package handler_test

import (
	"encoding/json"
	"net/http"
	"testing"
)

// ---------------------------------------------------------------------------
// Multiplayer Session API Tests
// ---------------------------------------------------------------------------

func TestMultiplayerSessionAPI(t *testing.T) {
	env := newTestEnv(t)

	// Setup: create a map with one round
	rr := env.doRequest(t, http.MethodPost, "/api/guide/maps/", map[string]string{"name": "MP Map"})
	if rr.Code != http.StatusCreated {
		t.Fatalf("create map: %d %s", rr.Code, rr.Body.String())
	}
	mapID := jsonField(t, rr, "id")

	rr = env.doRequest(t, http.MethodPost, "/api/guide/maps/"+mapID+"/rounds", map[string]any{
		"round_number": 1,
		"name":         "Round 1",
		"start_point":  json.RawMessage(startPointJSON),
		"end_point":    json.RawMessage(endPointJSON),
		"corridor":     json.RawMessage(corridorJSON),
	})
	if rr.Code != http.StatusCreated {
		t.Fatalf("create round: %d %s", rr.Code, rr.Body.String())
	}

	var sessionID string
	var sessionCode string

	t.Run("CreateSession_201", func(t *testing.T) {
		rr := env.doRequest(t, http.MethodPost, "/api/sessions/", map[string]any{
			"map_id":         mapID,
			"time_limit_sec": 300,
		})
		if rr.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
		}
		var sess map[string]any
		decodeJSON(t, rr, &sess)
		sessionID = sess["id"].(string)
		sessionCode = sess["code"].(string)
		if sessionID == "" {
			t.Fatal("session id should not be empty")
		}
		if sessionCode == "" {
			t.Fatal("session code should not be empty")
		}
		if sess["phase"] != "waiting" {
			t.Errorf("expected phase 'waiting', got %v", sess["phase"])
		}
	})

	t.Run("CreateSession_MissingMapID_400", func(t *testing.T) {
		rr := env.doRequest(t, http.MethodPost, "/api/sessions/", map[string]any{
			"time_limit_sec": 300,
		})
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("CreateSession_UnknownMap_404", func(t *testing.T) {
		rr := env.doRequest(t, http.MethodPost, "/api/sessions/", map[string]any{
			"map_id": "no-such-map",
		})
		if rr.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("GetSession_200", func(t *testing.T) {
		rr := env.doRequest(t, http.MethodGet, "/api/sessions/"+sessionID, nil)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}
		id := jsonField(t, rr, "id")
		if id != sessionID {
			t.Errorf("id mismatch: want %q, got %q", sessionID, id)
		}
	})

	t.Run("GetSession_NotFound_404", func(t *testing.T) {
		rr := env.doRequest(t, http.MethodGet, "/api/sessions/no-such-session", nil)
		if rr.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", rr.Code)
		}
	})

	t.Run("GetSessionByCode_200", func(t *testing.T) {
		rr := env.doRequest(t, http.MethodGet, "/api/sessions/code/"+sessionCode, nil)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
		id := jsonField(t, rr, "id")
		if id != sessionID {
			t.Errorf("id via code lookup: want %q, got %q", sessionID, id)
		}
	})

	t.Run("GetSessionByCode_NotFound_404", func(t *testing.T) {
		rr := env.doRequest(t, http.MethodGet, "/api/sessions/code/XXXXXX", nil)
		if rr.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", rr.Code)
		}
	})

	var teamID string

	t.Run("CreateTeam_201", func(t *testing.T) {
		rr := env.doRequest(t, http.MethodPost, "/api/sessions/"+sessionID+"/teams", map[string]string{
			"name":  "Avalanche",
			"color": "#EF4444",
		})
		if rr.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
		}
		teamID = jsonField(t, rr, "id")
		if teamID == "" {
			t.Fatal("team id should not be empty")
		}
	})

	t.Run("CreateTeam_MissingName_400", func(t *testing.T) {
		rr := env.doRequest(t, http.MethodPost, "/api/sessions/"+sessionID+"/teams", map[string]string{
			"color": "#3B82F6",
		})
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rr.Code)
		}
	})

	var playerID string

	t.Run("JoinSession_201", func(t *testing.T) {
		rr := env.doRequest(t, http.MethodPost, "/api/sessions/"+sessionID+"/join", map[string]string{
			"name": "Bode Miller",
		})
		if rr.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
		}
		playerID = jsonField(t, rr, "id")
		if playerID == "" {
			t.Fatal("player id should not be empty")
		}
	})

	t.Run("JoinSession_MissingName_400", func(t *testing.T) {
		rr := env.doRequest(t, http.MethodPost, "/api/sessions/"+sessionID+"/join", map[string]string{})
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rr.Code)
		}
	})

	t.Run("JoinSession_NotFound_404", func(t *testing.T) {
		rr := env.doRequest(t, http.MethodPost, "/api/sessions/no-such/join", map[string]string{"name": "Ghost"})
		if rr.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", rr.Code)
		}
	})

	t.Run("JoinTeam_200", func(t *testing.T) {
		rr := env.doRequest(t, http.MethodPost, "/api/sessions/"+sessionID+"/teams/"+teamID+"/join", map[string]string{
			"player_id": playerID,
		})
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
		var resp map[string]any
		decodeJSON(t, rr, &resp)
		if resp["status"] != "joined" {
			t.Errorf("expected status 'joined', got %v", resp["status"])
		}
	})

	t.Run("JoinTeam_MissingPlayerID_400", func(t *testing.T) {
		rr := env.doRequest(t, http.MethodPost, "/api/sessions/"+sessionID+"/teams/"+teamID+"/join", map[string]string{})
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rr.Code)
		}
	})

	t.Run("GetSession_IncludesTeamsAndPlayers", func(t *testing.T) {
		rr := env.doRequest(t, http.MethodGet, "/api/sessions/"+sessionID, nil)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}
		var sess map[string]any
		decodeJSON(t, rr, &sess)

		teams, _ := sess["teams"].([]any)
		if len(teams) != 1 {
			t.Errorf("expected 1 team in session, got %d", len(teams))
		}
		players, _ := sess["players"].([]any)
		if len(players) != 1 {
			t.Errorf("expected 1 player in session, got %d", len(players))
		}
	})

	t.Run("StartGame_NotEnoughTeams_400", func(t *testing.T) {
		// Multiplayer requires at least 2 teams; we only have 1
		rr := env.doRequest(t, http.MethodPost, "/api/sessions/"+sessionID+"/start", nil)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("JoinSession_GameAlreadyStarted_400", func(t *testing.T) {
		// Bootstrap a fresh session that has 2 teams to actually start
		rr2 := env.doRequest(t, http.MethodPost, "/api/sessions/solo", map[string]any{
			"map_id": mapID, "player_name": "Alice", "time_limit_sec": 300,
		})
		if rr2.Code != http.StatusCreated {
			t.Fatalf("create solo session: %d %s", rr2.Code, rr2.Body.String())
		}
		var soloResp struct {
			Session struct {
				ID string `json:"id"`
			} `json:"session"`
		}
		decodeJSON(t, rr2, &soloResp)
		startedID := soloResp.Session.ID

		// Start it
		env.doRequest(t, http.MethodPost, "/api/sessions/"+startedID+"/start", nil)

		// Joining an in-progress session should fail
		rr3 := env.doRequest(t, http.MethodPost, "/api/sessions/"+startedID+"/join", map[string]string{"name": "Latecomer"})
		if rr3.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", rr3.Code, rr3.Body.String())
		}
	})
}

// ---------------------------------------------------------------------------
// GetScores API Test
// ---------------------------------------------------------------------------

func TestGetScores(t *testing.T) {
	env := newTestEnv(t)

	// Setup map + round + solo session
	rr := env.doRequest(t, http.MethodPost, "/api/guide/maps/", map[string]string{"name": "Score Map"})
	mapID := jsonField(t, rr, "id")

	rr = env.doRequest(t, http.MethodPost, "/api/guide/maps/"+mapID+"/rounds", map[string]any{
		"round_number": 1,
		"name":         "Round 1",
		"start_point":  json.RawMessage(startPointJSON),
		"end_point":    json.RawMessage(endPointJSON),
		"corridor":     json.RawMessage(corridorJSON),
	})
	roundID := jsonField(t, rr, "id")

	rr = env.doRequest(t, http.MethodPost, "/api/sessions/solo", map[string]any{
		"map_id": mapID, "player_name": "Scorer", "time_limit_sec": 300,
	})
	var soloResp struct {
		Session struct {
			ID string `json:"id"`
		} `json:"session"`
		Team struct {
			ID string `json:"id"`
		} `json:"team"`
	}
	decodeJSON(t, rr, &soloResp)
	sessionID := soloResp.Session.ID
	teamID := soloResp.Team.ID

	env.doRequest(t, http.MethodPost, "/api/sessions/"+sessionID+"/start", nil)

	t.Run("GetScores_Empty_BeforeSubmission", func(t *testing.T) {
		rr := env.doRequest(t, http.MethodGet, "/api/sessions/"+sessionID+"/rounds/"+roundID+"/scores", nil)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
		var routes []any
		decodeJSON(t, rr, &routes)
		if len(routes) != 0 {
			t.Errorf("expected 0 routes before submission, got %d", len(routes))
		}
	})

	t.Run("GetScores_AfterSubmission_ReturnsRoute", func(t *testing.T) {
		// Submit a route
		env.doRequest(t, http.MethodPost, "/api/sessions/"+sessionID+"/rounds/"+roundID+"/submit", map[string]any{
			"team_id": teamID,
			"path":    json.RawMessage(routeLineJSON),
		})

		rr := env.doRequest(t, http.MethodGet, "/api/sessions/"+sessionID+"/rounds/"+roundID+"/scores", nil)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
		var routes []map[string]any
		decodeJSON(t, rr, &routes)
		if len(routes) != 1 {
			t.Fatalf("expected 1 route after submission, got %d", len(routes))
		}
		if routes[0]["team_id"] != teamID {
			t.Errorf("team_id: want %q, got %v", teamID, routes[0]["team_id"])
		}
		if routes[0]["score"] == nil {
			t.Error("score should not be nil")
		}
	})
}

// ---------------------------------------------------------------------------
// Update Map and Round Tests
// ---------------------------------------------------------------------------

func TestUpdateMapAndRound(t *testing.T) {
	env := newTestEnv(t)

	// Create a map
	rr := env.doRequest(t, http.MethodPost, "/api/guide/maps/", map[string]string{
		"name": "Original Name", "description": "Original Desc",
	})
	if rr.Code != http.StatusCreated {
		t.Fatalf("create map: %d", rr.Code)
	}
	mapID := jsonField(t, rr, "id")

	t.Run("UpdateMap_ChangesName", func(t *testing.T) {
		rr := env.doRequest(t, http.MethodPut, "/api/guide/maps/"+mapID, map[string]string{
			"name":        "New Name",
			"description": "New Desc",
		})
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("UpdateMap_NotFound_404", func(t *testing.T) {
		rr := env.doRequest(t, http.MethodPut, "/api/guide/maps/no-such-map", map[string]string{
			"name": "Whatever",
		})
		if rr.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", rr.Code)
		}
	})

	t.Run("GetMap_AfterUpdate_ReflectsChange", func(t *testing.T) {
		rr := env.doRequest(t, http.MethodGet, "/api/guide/maps/"+mapID, nil)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}
		name := jsonField(t, rr, "name")
		if name != "New Name" {
			t.Errorf("want 'New Name', got %q", name)
		}
	})

	// Create a round
	rr = env.doRequest(t, http.MethodPost, "/api/guide/maps/"+mapID+"/rounds", map[string]any{
		"round_number": 1,
		"name":         "Old Round Name",
		"start_point":  json.RawMessage(startPointJSON),
		"end_point":    json.RawMessage(endPointJSON),
		"corridor":     json.RawMessage(corridorJSON),
	})
	if rr.Code != http.StatusCreated {
		t.Fatalf("create round: %d %s", rr.Code, rr.Body.String())
	}
	roundID := jsonField(t, rr, "id")

	t.Run("UpdateRound_ChangesName", func(t *testing.T) {
		nameVal := "Updated Round Name"
		rr := env.doRequest(t, http.MethodPut, "/api/guide/maps/"+mapID+"/rounds/"+roundID, map[string]any{
			"name": &nameVal,
		})
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
		name := jsonField(t, rr, "name")
		if name != "Updated Round Name" {
			t.Errorf("want 'Updated Round Name', got %q", name)
		}
	})

	t.Run("UpdateRound_NotFound_404", func(t *testing.T) {
		nameVal := "Whatever"
		rr := env.doRequest(t, http.MethodPut, "/api/guide/maps/"+mapID+"/rounds/no-such-round", map[string]any{
			"name": &nameVal,
		})
		if rr.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", rr.Code)
		}
	})
}

// ---------------------------------------------------------------------------
// Multi-round Game: completes to 'finished'
// ---------------------------------------------------------------------------

func TestMultiRoundGameFinishes(t *testing.T) {
	env := newTestEnv(t)

	// Create map with 2 rounds
	rr := env.doRequest(t, http.MethodPost, "/api/guide/maps/", map[string]string{"name": "Two-Round Map"})
	mapID := jsonField(t, rr, "id")

	for _, round := range []struct {
		num  int
		name string
	}{
		{1, "Round 1"},
		{2, "Round 2"},
	} {
		rr = env.doRequest(t, http.MethodPost, "/api/guide/maps/"+mapID+"/rounds", map[string]any{
			"round_number": round.num,
			"name":         round.name,
			"start_point":  json.RawMessage(startPointJSON),
			"end_point":    json.RawMessage(endPointJSON),
			"corridor":     json.RawMessage(corridorJSON),
		})
		if rr.Code != http.StatusCreated {
			t.Fatalf("create round %d: %d %s", round.num, rr.Code, rr.Body.String())
		}
	}

	// Create solo session
	rr = env.doRequest(t, http.MethodPost, "/api/sessions/solo", map[string]any{
		"map_id": mapID, "player_name": "Pro Skier", "time_limit_sec": 300,
	})
	var soloResp struct {
		Session struct {
			ID string `json:"id"`
		} `json:"session"`
	}
	decodeJSON(t, rr, &soloResp)
	sessionID := soloResp.Session.ID

	// Start round 1
	rr = env.doRequest(t, http.MethodPost, "/api/sessions/"+sessionID+"/start", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("start round 1: %d %s", rr.Code, rr.Body.String())
	}
	var sess struct {
		Phase        string `json:"phase"`
		CurrentRound int    `json:"current_round"`
	}
	decodeJSON(t, rr, &sess)
	if sess.Phase != "playing" {
		t.Errorf("after round 1 start: expected phase 'playing', got %q", sess.Phase)
	}
	if sess.CurrentRound != 1 {
		t.Errorf("after round 1 start: expected current_round 1, got %d", sess.CurrentRound)
	}

	// Start round 2 (solo allows starting from 'playing')
	rr = env.doRequest(t, http.MethodPost, "/api/sessions/"+sessionID+"/start", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("start round 2: %d %s", rr.Code, rr.Body.String())
	}
	decodeJSON(t, rr, &sess)
	if sess.Phase != "playing" {
		t.Errorf("after round 2 start: expected phase 'playing', got %q", sess.Phase)
	}
	if sess.CurrentRound != 2 {
		t.Errorf("after round 2 start: expected current_round 2, got %d", sess.CurrentRound)
	}

	// Start again beyond last round → should finish
	rr = env.doRequest(t, http.MethodPost, "/api/sessions/"+sessionID+"/start", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("finish game: %d %s", rr.Code, rr.Body.String())
	}
	decodeJSON(t, rr, &sess)
	if sess.Phase != "finished" {
		t.Errorf("expected phase 'finished' after all rounds, got %q", sess.Phase)
	}

	// Further starts on a finished game should fail
	rr = env.doRequest(t, http.MethodPost, "/api/sessions/"+sessionID+"/start", nil)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 starting finished game, got %d: %s", rr.Code, rr.Body.String())
	}
}

// ---------------------------------------------------------------------------
// SubmitRoute error cases
// ---------------------------------------------------------------------------

// submitTestSetup creates a map+round+solo session in playing phase and returns
// the IDs needed for submit-route tests.
func submitTestSetup(t *testing.T, env *testEnv) (sessionID, roundID, teamID string) {
	t.Helper()
	rr := env.doRequest(t, http.MethodPost, "/api/guide/maps/", map[string]string{"name": "Submit Map"})
	if rr.Code != http.StatusCreated {
		t.Fatalf("create map: %d %s", rr.Code, rr.Body.String())
	}
	mapID := jsonField(t, rr, "id")

	rr = env.doRequest(t, http.MethodPost, "/api/guide/maps/"+mapID+"/rounds", map[string]any{
		"round_number": 1, "name": "R1",
		"start_point": json.RawMessage(startPointJSON),
		"end_point":   json.RawMessage(endPointJSON),
		"corridor":    json.RawMessage(corridorJSON),
	})
	if rr.Code != http.StatusCreated {
		t.Fatalf("create round: %d %s", rr.Code, rr.Body.String())
	}
	roundID = jsonField(t, rr, "id")

	rr = env.doRequest(t, http.MethodPost, "/api/sessions/solo", map[string]any{
		"map_id": mapID, "player_name": "Tester", "time_limit_sec": 300,
	})
	if rr.Code != http.StatusCreated {
		t.Fatalf("create solo session: %d %s", rr.Code, rr.Body.String())
	}
	var soloResp struct {
		Session struct {
			ID string `json:"id"`
		} `json:"session"`
		Team struct {
			ID string `json:"id"`
		} `json:"team"`
	}
	decodeJSON(t, rr, &soloResp)
	sessionID = soloResp.Session.ID
	teamID = soloResp.Team.ID

	rr = env.doRequest(t, http.MethodPost, "/api/sessions/"+sessionID+"/start", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("start game: %d %s", rr.Code, rr.Body.String())
	}
	return sessionID, roundID, teamID
}

func TestSubmitRoute_Errors(t *testing.T) {
	env := newTestEnv(t)

	t.Run("UnknownSession_404", func(t *testing.T) {
		rr := env.doRequest(t, http.MethodPost, "/api/sessions/no-such-session/rounds/no-such-round/submit", map[string]any{
			"team_id": "t1",
			"path":    json.RawMessage(routeLineJSON),
		})
		if rr.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("SessionNotPlaying_400", func(t *testing.T) {
		// Create a session but don't start it (phase=waiting)
		rr := env.doRequest(t, http.MethodPost, "/api/guide/maps/", map[string]string{"name": "Phase Map"})
		mapID := jsonField(t, rr, "id")
		env.doRequest(t, http.MethodPost, "/api/guide/maps/"+mapID+"/rounds", map[string]any{
			"round_number": 1, "name": "R1",
			"start_point": json.RawMessage(startPointJSON),
			"end_point":   json.RawMessage(endPointJSON),
			"corridor":    json.RawMessage(corridorJSON),
		})
		rr = env.doRequest(t, http.MethodPost, "/api/sessions/solo", map[string]any{
			"map_id": mapID, "player_name": "P", "time_limit_sec": 300,
		})
		var soloResp struct {
			Session struct {
				ID string `json:"id"`
			} `json:"session"`
			Team struct {
				ID string `json:"id"`
			} `json:"team"`
		}
		decodeJSON(t, rr, &soloResp)

		// Get the round ID
		rr2 := env.doRequest(t, http.MethodGet, "/api/guide/maps/"+mapID, nil)
		var m map[string]any
		decodeJSON(t, rr2, &m)
		rounds := m["rounds"].([]any)
		roundID := rounds[0].(map[string]any)["id"].(string)

		rr = env.doRequest(t, http.MethodPost, "/api/sessions/"+soloResp.Session.ID+"/rounds/"+roundID+"/submit", map[string]any{
			"team_id": soloResp.Team.ID,
			"path":    json.RawMessage(routeLineJSON),
		})
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for non-playing session, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("UnknownRound_404", func(t *testing.T) {
		sessionID, _, teamID := submitTestSetup(t, env)
		rr := env.doRequest(t, http.MethodPost, "/api/sessions/"+sessionID+"/rounds/no-such-round/submit", map[string]any{
			"team_id": teamID,
			"path":    json.RawMessage(routeLineJSON),
		})
		if rr.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("ForeignTeam_403", func(t *testing.T) {
		sessionID, roundID, _ := submitTestSetup(t, env)
		rr := env.doRequest(t, http.MethodPost, "/api/sessions/"+sessionID+"/rounds/"+roundID+"/submit", map[string]any{
			"team_id": "00000000-0000-0000-0000-000000000000",
			"path":    json.RawMessage(routeLineJSON),
		})
		if rr.Code != http.StatusForbidden {
			t.Fatalf("expected 403 for foreign team, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("DuplicateSubmission_409", func(t *testing.T) {
		sessionID, roundID, teamID := submitTestSetup(t, env)
		body := map[string]any{
			"team_id": teamID,
			"path":    json.RawMessage(routeLineJSON),
		}
		rr := env.doRequest(t, http.MethodPost, "/api/sessions/"+sessionID+"/rounds/"+roundID+"/submit", body)
		if rr.Code != http.StatusCreated {
			t.Fatalf("first submit: expected 201, got %d: %s", rr.Code, rr.Body.String())
		}
		rr = env.doRequest(t, http.MethodPost, "/api/sessions/"+sessionID+"/rounds/"+roundID+"/submit", body)
		if rr.Code != http.StatusConflict {
			t.Fatalf("second submit: expected 409, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("InvalidPath_400", func(t *testing.T) {
		sessionID, roundID, teamID := submitTestSetup(t, env)
		rr := env.doRequest(t, http.MethodPost, "/api/sessions/"+sessionID+"/rounds/"+roundID+"/submit", map[string]any{
			"team_id": teamID,
			"path":    json.RawMessage(`"not a linestring"`),
		})
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for invalid path, got %d: %s", rr.Code, rr.Body.String())
		}
	})
}
