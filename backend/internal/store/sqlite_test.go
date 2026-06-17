package store_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/paulmach/orb"

	"github.com/colinbradley/sluff/internal/model"
	"github.com/colinbradley/sluff/internal/store"
)

// newTestStore opens a real, file-backed SQLite database in a temporary
// directory so tests exercise the full persistence path (WAL, foreign keys,
// migrations) rather than an in-memory shortcut.
func newTestStore(t *testing.T) *store.SQLiteStore {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	s, err := store.NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("newTestStore: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func newMap(name string) *model.GameMap {
	now := time.Now()
	return &model.GameMap{
		ID:          "map-" + name,
		Name:        name,
		Description: "desc " + name,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

func newRound(mapID, id string) *model.Round {
	return &model.Round{
		ID:          id,
		MapID:       mapID,
		RoundNumber: 1,
		Name:        "Round 1",
		StartPoint:  orb.Point{-111.58, 40.59},
		EndPoint:    orb.Point{-111.56, 40.61},
		Corridor: orb.Polygon{orb.Ring{
			{-111.60, 40.58},
			{-111.54, 40.58},
			{-111.54, 40.62},
			{-111.60, 40.62},
			{-111.60, 40.58},
		}},
	}
}

func newSession(mapID, id string) *model.Session {
	return &model.Session{
		ID:           id,
		MapID:        mapID,
		Code:         "TESTCD",
		Phase:        model.PhaseWaiting,
		CurrentRound: 0,
		TimeLimitSec: 300,
		CreatedAt:    time.Now(),
	}
}

// ---------------------------------------------------------------------------
// Maps
// ---------------------------------------------------------------------------

func TestStore_Maps(t *testing.T) {
	s := newTestStore(t)

	m := newMap("Alpha")

	t.Run("CreateMap and GetMap round-trip", func(t *testing.T) {
		if err := s.CreateMap(m); err != nil {
			t.Fatalf("CreateMap: %v", err)
		}
		got, err := s.GetMap(m.ID)
		if err != nil {
			t.Fatalf("GetMap: %v", err)
		}
		if got == nil {
			t.Fatal("expected map, got nil")
		}
		if got.Name != m.Name {
			t.Errorf("name: want %q, got %q", m.Name, got.Name)
		}
		if got.Description != m.Description {
			t.Errorf("description: want %q, got %q", m.Description, got.Description)
		}
	})

	t.Run("GetMap returns nil for unknown id", func(t *testing.T) {
		got, err := s.GetMap("no-such-id")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != nil {
			t.Errorf("expected nil, got %+v", got)
		}
	})

	t.Run("UpdateMap persists name change", func(t *testing.T) {
		m.Name = "Alpha Updated"
		if err := s.UpdateMap(m); err != nil {
			t.Fatalf("UpdateMap: %v", err)
		}
		got, _ := s.GetMap(m.ID)
		if got.Name != "Alpha Updated" {
			t.Errorf("want 'Alpha Updated', got %q", got.Name)
		}
	})

	t.Run("DeleteMap removes it", func(t *testing.T) {
		if err := s.DeleteMap(m.ID); err != nil {
			t.Fatalf("DeleteMap: %v", err)
		}
		got, err := s.GetMap(m.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != nil {
			t.Error("expected nil after delete")
		}
	})
}

// ---------------------------------------------------------------------------
// Rounds
// ---------------------------------------------------------------------------

func TestStore_Rounds(t *testing.T) {
	s := newTestStore(t)

	m := newMap("RoundMap")
	if err := s.CreateMap(m); err != nil {
		t.Fatalf("setup: CreateMap: %v", err)
	}

	r := newRound(m.ID, "round-1")

	t.Run("CreateRound and GetRound round-trip", func(t *testing.T) {
		if err := s.CreateRound(r); err != nil {
			t.Fatalf("CreateRound: %v", err)
		}
		got, err := s.GetRound(r.ID)
		if err != nil {
			t.Fatalf("GetRound: %v", err)
		}
		if got == nil {
			t.Fatal("expected round, got nil")
		}
		if got.Name != r.Name {
			t.Errorf("name: want %q, got %q", r.Name, got.Name)
		}
		if got.StartPoint[0] != r.StartPoint[0] || got.StartPoint[1] != r.StartPoint[1] {
			t.Errorf("start point: want %v, got %v", r.StartPoint, got.StartPoint)
		}
		if got.EndPoint[0] != r.EndPoint[0] || got.EndPoint[1] != r.EndPoint[1] {
			t.Errorf("end point: want %v, got %v", r.EndPoint, got.EndPoint)
		}
		if len(got.Corridor) == 0 || len(got.Corridor[0]) == 0 {
			t.Error("corridor should not be empty after round-trip")
		}
	})

	t.Run("GetRound returns nil for unknown id", func(t *testing.T) {
		got, err := s.GetRound("no-such-round")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != nil {
			t.Errorf("expected nil, got %+v", got)
		}
	})

	t.Run("GetRoundsByMap returns correct rounds", func(t *testing.T) {
		rounds, err := s.GetRoundsByMap(m.ID)
		if err != nil {
			t.Fatalf("GetRoundsByMap: %v", err)
		}
		if len(rounds) != 1 {
			t.Fatalf("expected 1 round, got %d", len(rounds))
		}
		if rounds[0].ID != r.ID {
			t.Errorf("id: want %q, got %q", r.ID, rounds[0].ID)
		}
	})

	t.Run("GetRoundsByMap returns empty slice for unknown map", func(t *testing.T) {
		rounds, err := s.GetRoundsByMap("no-such-map")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(rounds) != 0 {
			t.Errorf("expected empty, got %d rounds", len(rounds))
		}
	})

	t.Run("UpdateRound persists name change", func(t *testing.T) {
		r.Name = "Updated Round"
		r.RoundNumber = 2
		if err := s.UpdateRound(r); err != nil {
			t.Fatalf("UpdateRound: %v", err)
		}
		got, _ := s.GetRound(r.ID)
		if got.Name != "Updated Round" {
			t.Errorf("want 'Updated Round', got %q", got.Name)
		}
		if got.RoundNumber != 2 {
			t.Errorf("want round_number 2, got %d", got.RoundNumber)
		}
	})

	t.Run("DeleteRound removes it", func(t *testing.T) {
		if err := s.DeleteRound(r.ID); err != nil {
			t.Fatalf("DeleteRound: %v", err)
		}
		got, err := s.GetRound(r.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != nil {
			t.Error("expected nil after delete")
		}
	})

	t.Run("GetMap includes rounds", func(t *testing.T) {
		// Create a fresh round after the delete above
		r2 := newRound(m.ID, "round-2")
		r2.RoundNumber = 1
		_ = s.CreateRound(r2)

		gm, err := s.GetMap(m.ID)
		if err != nil {
			t.Fatalf("GetMap: %v", err)
		}
		if len(gm.Rounds) != 1 {
			t.Errorf("expected 1 round in map, got %d", len(gm.Rounds))
		}
	})
}

// ---------------------------------------------------------------------------
// Sessions, Teams, Players
// ---------------------------------------------------------------------------

func TestStore_Sessions(t *testing.T) {
	s := newTestStore(t)

	m := newMap("SessionMap")
	_ = s.CreateMap(m)

	sess := newSession(m.ID, "sess-1")

	t.Run("CreateSession and GetSession round-trip", func(t *testing.T) {
		if err := s.CreateSession(sess); err != nil {
			t.Fatalf("CreateSession: %v", err)
		}
		got, err := s.GetSession(sess.ID)
		if err != nil {
			t.Fatalf("GetSession: %v", err)
		}
		if got == nil {
			t.Fatal("expected session, got nil")
		}
		if got.Phase != model.PhaseWaiting {
			t.Errorf("phase: want %q, got %q", model.PhaseWaiting, got.Phase)
		}
		if got.MapID != m.ID {
			t.Errorf("map_id: want %q, got %q", m.ID, got.MapID)
		}
	})

	t.Run("GetSession returns nil for unknown id", func(t *testing.T) {
		got, err := s.GetSession("no-such-session")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != nil {
			t.Errorf("expected nil, got %+v", got)
		}
	})

	t.Run("GetSessionByCode returns correct session", func(t *testing.T) {
		got, err := s.GetSessionByCode(sess.Code)
		if err != nil {
			t.Fatalf("GetSessionByCode: %v", err)
		}
		if got == nil {
			t.Fatal("expected session, got nil")
		}
		if got.ID != sess.ID {
			t.Errorf("id: want %q, got %q", sess.ID, got.ID)
		}
	})

	t.Run("GetSessionByCode returns nil for unknown code", func(t *testing.T) {
		got, err := s.GetSessionByCode("XXXXXX")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != nil {
			t.Error("expected nil for unknown code")
		}
	})

	t.Run("UpdateSession persists phase and round", func(t *testing.T) {
		sess.Phase = model.PhasePlaying
		sess.CurrentRound = 1
		if err := s.UpdateSession(sess); err != nil {
			t.Fatalf("UpdateSession: %v", err)
		}
		got, _ := s.GetSession(sess.ID)
		if got.Phase != model.PhasePlaying {
			t.Errorf("phase: want %q, got %q", model.PhasePlaying, got.Phase)
		}
		if got.CurrentRound != 1 {
			t.Errorf("current_round: want 1, got %d", got.CurrentRound)
		}
	})
}

func TestStore_Teams(t *testing.T) {
	s := newTestStore(t)

	m := newMap("TeamMap")
	_ = s.CreateMap(m)
	sess := newSession(m.ID, "sess-teams")
	_ = s.CreateSession(sess)

	team := &model.Team{
		ID:        "team-1",
		SessionID: sess.ID,
		Name:      "Red Team",
		Color:     "#EF4444",
	}

	t.Run("CreateTeam and GetTeamsBySession", func(t *testing.T) {
		if err := s.CreateTeam(team); err != nil {
			t.Fatalf("CreateTeam: %v", err)
		}
		teams, err := s.GetTeamsBySession(sess.ID)
		if err != nil {
			t.Fatalf("GetTeamsBySession: %v", err)
		}
		if len(teams) != 1 {
			t.Fatalf("expected 1 team, got %d", len(teams))
		}
		if teams[0].Name != "Red Team" {
			t.Errorf("name: want 'Red Team', got %q", teams[0].Name)
		}
	})

	t.Run("GetTeamsBySession returns empty for unknown session", func(t *testing.T) {
		teams, err := s.GetTeamsBySession("no-such-session")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(teams) != 0 {
			t.Errorf("expected empty, got %d teams", len(teams))
		}
	})

	t.Run("GetSession includes teams", func(t *testing.T) {
		got, _ := s.GetSession(sess.ID)
		if len(got.Teams) != 1 {
			t.Errorf("expected 1 team in session, got %d", len(got.Teams))
		}
	})
}

func TestStore_Players(t *testing.T) {
	s := newTestStore(t)

	m := newMap("PlayerMap")
	_ = s.CreateMap(m)
	sess := newSession(m.ID, "sess-players")
	_ = s.CreateSession(sess)
	team := &model.Team{ID: "team-p", SessionID: sess.ID, Name: "Blue", Color: "#3B82F6"}
	_ = s.CreateTeam(team)

	player := &model.Player{
		ID:        "player-1",
		SessionID: sess.ID,
		TeamID:    team.ID,
		Name:      "Alice",
	}

	t.Run("CreatePlayer and GetPlayer round-trip", func(t *testing.T) {
		if err := s.CreatePlayer(player); err != nil {
			t.Fatalf("CreatePlayer: %v", err)
		}
		got, err := s.GetPlayer(player.ID)
		if err != nil {
			t.Fatalf("GetPlayer: %v", err)
		}
		if got == nil {
			t.Fatal("expected player, got nil")
		}
		if got.Name != "Alice" {
			t.Errorf("name: want 'Alice', got %q", got.Name)
		}
		if got.TeamID != team.ID {
			t.Errorf("team_id: want %q, got %q", team.ID, got.TeamID)
		}
	})

	t.Run("GetPlayer returns nil for unknown id", func(t *testing.T) {
		got, err := s.GetPlayer("no-such-player")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != nil {
			t.Error("expected nil for unknown player")
		}
	})

	t.Run("GetPlayersBySession returns correct players", func(t *testing.T) {
		players, err := s.GetPlayersBySession(sess.ID)
		if err != nil {
			t.Fatalf("GetPlayersBySession: %v", err)
		}
		if len(players) != 1 {
			t.Fatalf("expected 1 player, got %d", len(players))
		}
	})

	t.Run("UpdatePlayerTeam changes team", func(t *testing.T) {
		newTeam := &model.Team{ID: "team-new", SessionID: sess.ID, Name: "Green", Color: "#10B981"}
		_ = s.CreateTeam(newTeam)

		if err := s.UpdatePlayerTeam(player.ID, newTeam.ID); err != nil {
			t.Fatalf("UpdatePlayerTeam: %v", err)
		}
		got, _ := s.GetPlayer(player.ID)
		if got.TeamID != newTeam.ID {
			t.Errorf("team_id after update: want %q, got %q", newTeam.ID, got.TeamID)
		}
	})

	t.Run("Player without team is stored with empty team_id", func(t *testing.T) {
		p := &model.Player{ID: "player-no-team", SessionID: sess.ID, TeamID: "", Name: "Bob"}
		if err := s.CreatePlayer(p); err != nil {
			t.Fatalf("CreatePlayer without team: %v", err)
		}
		got, _ := s.GetPlayer(p.ID)
		if got.TeamID != "" {
			t.Errorf("expected empty team_id, got %q", got.TeamID)
		}
	})
}

// ---------------------------------------------------------------------------
// Routes
// ---------------------------------------------------------------------------

func TestStore_Routes(t *testing.T) {
	s := newTestStore(t)

	m := newMap("RouteMap")
	_ = s.CreateMap(m)
	r := newRound(m.ID, "round-route")
	_ = s.CreateRound(r)
	sess := newSession(m.ID, "sess-routes")
	_ = s.CreateSession(sess)
	team := &model.Team{ID: "team-route", SessionID: sess.ID, Name: "Solo", Color: "#3B82F6"}
	_ = s.CreateTeam(team)

	score := 850.0
	details := &model.ScoreDetails{
		TotalPoints:       100,
		PointsInCorridor:  95,
		PercentInCorridor: 95.0,
		RouteLengthKm:     3.5,
		MaxDeviationM:     50.0,
		ConnectsStart:     true,
		ConnectsEnd:       true,
		FinalScore:        850.0,
	}
	now := time.Now()
	route := &model.TeamRoute{
		ID:          "route-1",
		SessionID:   sess.ID,
		RoundID:     r.ID,
		TeamID:      team.ID,
		Path:        `{"type":"LineString","coordinates":[[-111.58,40.59],[-111.57,40.60],[-111.56,40.61]]}`,
		Score:       &score,
		Details:     details,
		SubmittedAt: &now,
	}

	t.Run("CreateTeamRoute and GetTeamRoute round-trip", func(t *testing.T) {
		if err := s.CreateTeamRoute(route); err != nil {
			t.Fatalf("CreateTeamRoute: %v", err)
		}
		got, err := s.GetTeamRoute(r.ID, team.ID)
		if err != nil {
			t.Fatalf("GetTeamRoute: %v", err)
		}
		if got == nil {
			t.Fatal("expected route, got nil")
		}
		if *got.Score != 850.0 {
			t.Errorf("score: want 850.0, got %f", *got.Score)
		}
		if got.Details == nil {
			t.Fatal("expected details, got nil")
		}
		if !got.Details.ConnectsStart {
			t.Error("connects_start should be true")
		}
		if got.Details.PercentInCorridor != 95.0 {
			t.Errorf("percent_in_corridor: want 95.0, got %f", got.Details.PercentInCorridor)
		}
	})

	t.Run("GetTeamRoute returns nil for unknown round/team", func(t *testing.T) {
		got, err := s.GetTeamRoute("no-round", "no-team")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != nil {
			t.Error("expected nil")
		}
	})

	t.Run("GetRoutesByRound returns all team routes", func(t *testing.T) {
		routes, err := s.GetRoutesByRound(r.ID)
		if err != nil {
			t.Fatalf("GetRoutesByRound: %v", err)
		}
		if len(routes) != 1 {
			t.Fatalf("expected 1 route, got %d", len(routes))
		}
	})

	t.Run("UpdateTeamRouteScore changes score and details", func(t *testing.T) {
		newDetails := model.ScoreDetails{FinalScore: 950.0, ConnectsStart: true, ConnectsEnd: true}
		if err := s.UpdateTeamRouteScore(route.ID, 950.0, newDetails.ToJSON()); err != nil {
			t.Fatalf("UpdateTeamRouteScore: %v", err)
		}
		got, _ := s.GetTeamRoute(r.ID, team.ID)
		if *got.Score != 950.0 {
			t.Errorf("updated score: want 950.0, got %f", *got.Score)
		}
		if got.Details.FinalScore != 950.0 {
			t.Errorf("updated details.final_score: want 950.0, got %f", got.Details.FinalScore)
		}
	})

	t.Run("GetRoutesByRound returns empty for unknown round", func(t *testing.T) {
		routes, err := s.GetRoutesByRound("no-such-round")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(routes) != 0 {
			t.Errorf("expected empty, got %d routes", len(routes))
		}
	})
}
