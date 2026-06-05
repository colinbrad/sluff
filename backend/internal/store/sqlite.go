package store

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3" // registers the sqlite3 driver with database/sql
	"golang.org/x/crypto/bcrypt"

	"github.com/google/uuid"

	"github.com/colinbradley/sluff/internal/model"
	"github.com/paulmach/orb/geojson"
)

// SQLiteStore is a Store backed by a SQLite database connection.
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore opens (or creates) a SQLite database at dbPath, runs the
// schema and alter migrations, and seeds a default guide if none exist.
func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	if _, err := db.Exec(schema); err != nil {
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	for _, m := range alterMigrations {
		if _, err := db.Exec(m); err != nil && !strings.Contains(err.Error(), "duplicate column") {
			return nil, fmt.Errorf("run migration: %w", err)
		}
	}

	if err := seedDefaultGuide(db); err != nil {
		return nil, fmt.Errorf("seed default guide: %w", err)
	}

	return &SQLiteStore{db: db}, nil
}

// seedDefaultGuide creates a default guide account if none exist, and assigns
// all existing un-owned maps and sessions to it.
func seedDefaultGuide(db *sql.DB) error {
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM guides").Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	username := os.Getenv("DEFAULT_GUIDE_USERNAME")
	if username == "" {
		username = "admin"
	}
	password := os.Getenv("DEFAULT_GUIDE_PASSWORD")
	if password == "" {
		password = "sluff"
	}
	if password == "sluff" && os.Getenv("ALLOW_DEFAULT_GUIDE") != "true" {
		slog.Info("skipping default guide seed; set DEFAULT_GUIDE_PASSWORD or ALLOW_DEFAULT_GUIDE=true to seed")
		return nil
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	id := uuid.New().String()
	if _, err := db.Exec(
		"INSERT INTO guides (id, username, password_hash, created_at) VALUES (?, ?, ?, CURRENT_TIMESTAMP)",
		id, username, string(hash),
	); err != nil {
		return fmt.Errorf("insert guide: %w", err)
	}

	if _, err := db.Exec("UPDATE game_maps SET guide_id = ? WHERE guide_id IS NULL", id); err != nil {
		return fmt.Errorf("assign existing maps to default guide: %w", err)
	}
	if _, err := db.Exec("UPDATE sessions SET guide_id = ? WHERE guide_id IS NULL", id); err != nil {
		return fmt.Errorf("assign existing sessions to default guide: %w", err)
	}

	slog.Info("created default guide", "username", username)
	return nil
}

// Close closes the underlying database connection.
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

// --- Guides ---

// CreateGuide inserts a guide row.
func (s *SQLiteStore) CreateGuide(g *model.Guide) error {
	_, err := s.db.Exec(
		"INSERT INTO guides (id, username, password_hash, created_at) VALUES (?, ?, ?, ?)",
		g.ID, g.Username, g.PasswordHash, g.CreatedAt,
	)
	return err
}

// GetGuideByUsername returns a guide by username, or (nil, nil) if not found.
func (s *SQLiteStore) GetGuideByUsername(username string) (*model.Guide, error) {
	g := &model.Guide{}
	err := s.db.QueryRow("SELECT id, username, password_hash, created_at FROM guides WHERE username = ?", username).
		Scan(&g.ID, &g.Username, &g.PasswordHash, &g.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return g, err
}

// GetGuideByID returns a guide by ID, or (nil, nil) if not found.
func (s *SQLiteStore) GetGuideByID(id string) (*model.Guide, error) {
	g := &model.Guide{}
	err := s.db.QueryRow("SELECT id, username, password_hash, created_at FROM guides WHERE id = ?", id).
		Scan(&g.ID, &g.Username, &g.PasswordHash, &g.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return g, err
}

// --- Maps ---

// CreateMap inserts a new map row.
func (s *SQLiteStore) CreateMap(m *model.GameMap) error {
	var guideID any
	if m.GuideID != "" {
		guideID = m.GuideID
	}
	_, err := s.db.Exec(
		"INSERT INTO game_maps (id, name, description, guide_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
		m.ID, m.Name, m.Description, guideID, m.CreatedAt, m.UpdatedAt,
	)
	return err
}

// GetMap returns a map (with its rounds) by ID, or (nil, nil) if not found.
func (s *SQLiteStore) GetMap(id string) (*model.GameMap, error) {
	m := &model.GameMap{}
	var guideID sql.NullString
	err := s.db.QueryRow("SELECT id, name, description, guide_id, created_at, updated_at FROM game_maps WHERE id = ?", id).
		Scan(&m.ID, &m.Name, &m.Description, &guideID, &m.CreatedAt, &m.UpdatedAt)
	if guideID.Valid {
		m.GuideID = guideID.String
	}
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	rounds, err := s.GetRoundsByMap(id)
	if err != nil {
		return nil, err
	}
	m.Rounds = rounds
	return m, nil
}

// ListMapsByGuide returns all maps owned by the given guide, newest first.
func (s *SQLiteStore) ListMapsByGuide(guideID string) ([]model.GameMap, error) {
	rows, err := s.db.Query(
		"SELECT id, name, description, guide_id, created_at, updated_at FROM game_maps WHERE guide_id = ? ORDER BY created_at DESC",
		guideID,
	)
	if err != nil {
		return nil, err
	}

	var maps []model.GameMap
	for rows.Next() {
		var m model.GameMap
		var gid sql.NullString
		if err := rows.Scan(&m.ID, &m.Name, &m.Description, &gid, &m.CreatedAt, &m.UpdatedAt); err != nil {
			_ = rows.Close()
			return nil, err
		}
		if gid.Valid {
			m.GuideID = gid.String
		}
		maps = append(maps, m)
	}
	_ = rows.Close()

	for i := range maps {
		rounds, err := s.GetRoundsByMap(maps[i].ID)
		if err != nil {
			return nil, err
		}
		maps[i].Rounds = rounds
	}
	return maps, nil
}

// ListAllMaps returns every map in the database, oldest first.
func (s *SQLiteStore) ListAllMaps() ([]model.GameMap, error) {
	rows, err := s.db.Query(
		"SELECT id, name, description, guide_id, created_at, updated_at FROM game_maps ORDER BY created_at ASC",
	)
	if err != nil {
		return nil, err
	}

	var maps []model.GameMap
	for rows.Next() {
		var m model.GameMap
		var gid sql.NullString
		if err := rows.Scan(&m.ID, &m.Name, &m.Description, &gid, &m.CreatedAt, &m.UpdatedAt); err != nil {
			_ = rows.Close()
			return nil, err
		}
		if gid.Valid {
			m.GuideID = gid.String
		}
		maps = append(maps, m)
	}
	_ = rows.Close()

	for i := range maps {
		rounds, err := s.GetRoundsByMap(maps[i].ID)
		if err != nil {
			return nil, err
		}
		maps[i].Rounds = rounds
	}
	return maps, nil
}

// UpdateMap persists the map's name and description and bumps updated_at.
func (s *SQLiteStore) UpdateMap(m *model.GameMap) error {
	m.UpdatedAt = time.Now()
	_, err := s.db.Exec(
		"UPDATE game_maps SET name = ?, description = ?, updated_at = ? WHERE id = ?",
		m.Name, m.Description, m.UpdatedAt, m.ID,
	)
	return err
}

// DeleteMap removes a map row; cascades remove its rounds.
func (s *SQLiteStore) DeleteMap(id string) error {
	_, err := s.db.Exec("DELETE FROM game_maps WHERE id = ?", id)
	return err
}

// --- Rounds ---

// CreateRound inserts a new round, encoding its geometries as GeoJSON.
func (s *SQLiteStore) CreateRound(r *model.Round) error {
	startJSON, _ := geojson.NewGeometry(r.StartPoint).MarshalJSON()
	endJSON, _ := geojson.NewGeometry(r.EndPoint).MarshalJSON()
	corrJSON, _ := geojson.NewGeometry(r.Corridor).MarshalJSON()
	noGoJSON := model.NoGoZonesToJSON(r.NoGoZones)

	_, err := s.db.Exec(
		"INSERT INTO rounds (id, map_id, round_number, name, start_point, end_point, corridor, no_go_zones) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		r.ID, r.MapID, r.RoundNumber, r.Name, string(startJSON), string(endJSON), string(corrJSON), noGoJSON,
	)
	return err
}

// GetRound returns a round by ID, or (nil, nil) if not found.
func (s *SQLiteStore) GetRound(id string) (*model.Round, error) {
	r := &model.Round{}
	var startJSON, endJSON, corrJSON, noGoJSON string
	err := s.db.QueryRow("SELECT id, map_id, round_number, name, start_point, end_point, corridor, no_go_zones FROM rounds WHERE id = ?", id).
		Scan(&r.ID, &r.MapID, &r.RoundNumber, &r.Name, &startJSON, &endJSON, &corrJSON, &noGoJSON)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	r.StartPoint, r.EndPoint, r.Corridor, err = model.RoundFromJSON(startJSON, endJSON, corrJSON)
	if err != nil {
		return nil, fmt.Errorf("parse round geometry: %w", err)
	}
	r.NoGoZones, err = model.NoGoZonesFromJSON(noGoJSON)
	if err != nil {
		return nil, fmt.Errorf("parse no-go zones: %w", err)
	}
	return r, nil
}

// GetRoundsByMap returns the rounds of a map ordered by round_number.
func (s *SQLiteStore) GetRoundsByMap(mapID string) ([]model.Round, error) {
	rows, err := s.db.Query("SELECT id, map_id, round_number, name, start_point, end_point, corridor, no_go_zones FROM rounds WHERE map_id = ? ORDER BY round_number", mapID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var rounds []model.Round
	for rows.Next() {
		var r model.Round
		var startJSON, endJSON, corrJSON, noGoJSON string
		if err := rows.Scan(&r.ID, &r.MapID, &r.RoundNumber, &r.Name, &startJSON, &endJSON, &corrJSON, &noGoJSON); err != nil {
			return nil, err
		}
		r.StartPoint, r.EndPoint, r.Corridor, err = model.RoundFromJSON(startJSON, endJSON, corrJSON)
		if err != nil {
			return nil, fmt.Errorf("parse round geometry: %w", err)
		}
		r.NoGoZones, err = model.NoGoZonesFromJSON(noGoJSON)
		if err != nil {
			return nil, fmt.Errorf("parse no-go zones: %w", err)
		}
		rounds = append(rounds, r)
	}
	return rounds, rows.Err()
}

// UpdateRound rewrites a round's mutable fields and geometries.
func (s *SQLiteStore) UpdateRound(r *model.Round) error {
	startJSON, _ := geojson.NewGeometry(r.StartPoint).MarshalJSON()
	endJSON, _ := geojson.NewGeometry(r.EndPoint).MarshalJSON()
	corrJSON, _ := geojson.NewGeometry(r.Corridor).MarshalJSON()
	noGoJSON := model.NoGoZonesToJSON(r.NoGoZones)

	_, err := s.db.Exec(
		"UPDATE rounds SET round_number = ?, name = ?, start_point = ?, end_point = ?, corridor = ?, no_go_zones = ? WHERE id = ?",
		r.RoundNumber, r.Name, string(startJSON), string(endJSON), string(corrJSON), noGoJSON, r.ID,
	)
	return err
}

// DeleteRound removes a round row.
func (s *SQLiteStore) DeleteRound(id string) error {
	_, err := s.db.Exec("DELETE FROM rounds WHERE id = ?", id)
	return err
}

// --- Sessions ---

// CreateSession inserts a new session row.
func (s *SQLiteStore) CreateSession(sess *model.Session) error {
	var guideID any
	if sess.GuideID != "" {
		guideID = sess.GuideID
	}
	_, err := s.db.Exec(
		"INSERT INTO sessions (id, map_id, guide_id, code, phase, current_round, time_limit_sec, is_solo, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
		sess.ID, sess.MapID, guideID, sess.Code, sess.Phase, sess.CurrentRound, sess.TimeLimitSec, sess.IsSolo, sess.CreatedAt,
	)
	return err
}

// GetSession returns a session (with teams and players) by ID, or (nil, nil)
// if not found.
func (s *SQLiteStore) GetSession(id string) (*model.Session, error) {
	sess := &model.Session{}
	var guideID sql.NullString
	err := s.db.QueryRow("SELECT id, map_id, guide_id, code, phase, current_round, time_limit_sec, is_solo, created_at FROM sessions WHERE id = ?", id).
		Scan(&sess.ID, &sess.MapID, &guideID, &sess.Code, &sess.Phase, &sess.CurrentRound, &sess.TimeLimitSec, &sess.IsSolo, &sess.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if guideID.Valid {
		sess.GuideID = guideID.String
	}

	teams, err := s.GetTeamsBySession(id)
	if err != nil {
		return nil, err
	}
	sess.Teams = teams

	players, err := s.GetPlayersBySession(id)
	if err != nil {
		return nil, err
	}
	sess.Players = players
	return sess, nil
}

// GetSessionByCode returns a session by its short join code.
func (s *SQLiteStore) GetSessionByCode(code string) (*model.Session, error) {
	var id string
	err := s.db.QueryRow("SELECT id FROM sessions WHERE code = ?", code).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return s.GetSession(id)
}

// UpdateSession persists the session's phase and current round.
func (s *SQLiteStore) UpdateSession(sess *model.Session) error {
	_, err := s.db.Exec(
		"UPDATE sessions SET phase = ?, current_round = ? WHERE id = ?",
		sess.Phase, sess.CurrentRound, sess.ID,
	)
	return err
}

// --- Teams ---

// CreateTeam inserts a new team row.
func (s *SQLiteStore) CreateTeam(t *model.Team) error {
	_, err := s.db.Exec(
		"INSERT INTO teams (id, session_id, name, color) VALUES (?, ?, ?, ?)",
		t.ID, t.SessionID, t.Name, t.Color,
	)
	return err
}

// GetTeamsBySession returns the teams in a session.
func (s *SQLiteStore) GetTeamsBySession(sessionID string) ([]model.Team, error) {
	rows, err := s.db.Query("SELECT id, session_id, name, color FROM teams WHERE session_id = ?", sessionID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var teams []model.Team
	for rows.Next() {
		var t model.Team
		if err := rows.Scan(&t.ID, &t.SessionID, &t.Name, &t.Color); err != nil {
			return nil, err
		}
		teams = append(teams, t)
	}
	return teams, rows.Err()
}

// --- Players ---

// CreatePlayer inserts a new player row.
func (s *SQLiteStore) CreatePlayer(p *model.Player) error {
	var teamID any
	if p.TeamID != "" {
		teamID = p.TeamID
	}
	_, err := s.db.Exec(
		"INSERT INTO players (id, session_id, team_id, name) VALUES (?, ?, ?, ?)",
		p.ID, p.SessionID, teamID, p.Name,
	)
	return err
}

// GetPlayer returns a player by ID, or (nil, nil) if not found.
func (s *SQLiteStore) GetPlayer(id string) (*model.Player, error) {
	p := &model.Player{}
	var teamID sql.NullString
	err := s.db.QueryRow("SELECT id, session_id, team_id, name FROM players WHERE id = ?", id).
		Scan(&p.ID, &p.SessionID, &teamID, &p.Name)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if teamID.Valid {
		p.TeamID = teamID.String
	}
	return p, nil
}

// GetPlayersBySession returns the players in a session.
func (s *SQLiteStore) GetPlayersBySession(sessionID string) ([]model.Player, error) {
	rows, err := s.db.Query("SELECT id, session_id, team_id, name FROM players WHERE session_id = ?", sessionID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var players []model.Player
	for rows.Next() {
		var p model.Player
		var teamID sql.NullString
		if err := rows.Scan(&p.ID, &p.SessionID, &teamID, &p.Name); err != nil {
			return nil, err
		}
		if teamID.Valid {
			p.TeamID = teamID.String
		}
		players = append(players, p)
	}
	return players, rows.Err()
}

// UpdatePlayerTeam assigns a player to a team.
func (s *SQLiteStore) UpdatePlayerTeam(playerID, teamID string) error {
	_, err := s.db.Exec("UPDATE players SET team_id = ? WHERE id = ?", teamID, playerID)
	return err
}

// --- Routes ---

// CreateTeamRoute persists a team's submitted route and score breakdown.
func (s *SQLiteStore) CreateTeamRoute(r *model.TeamRoute) error {
	var detailsJSON *string
	if r.Details != nil {
		j := r.Details.ToJSON()
		detailsJSON = &j
	}
	_, err := s.db.Exec(
		"INSERT INTO team_routes (id, session_id, round_id, team_id, path, score, details, submitted_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		r.ID, r.SessionID, r.RoundID, r.TeamID, r.Path, r.Score, detailsJSON, r.SubmittedAt,
	)
	return err
}

// GetTeamRoute returns a team's submission for a round, or (nil, nil) if none.
func (s *SQLiteStore) GetTeamRoute(roundID, teamID string) (*model.TeamRoute, error) {
	r := &model.TeamRoute{}
	var detailsJSON sql.NullString
	err := s.db.QueryRow("SELECT id, session_id, round_id, team_id, path, score, details, submitted_at FROM team_routes WHERE round_id = ? AND team_id = ?", roundID, teamID).
		Scan(&r.ID, &r.SessionID, &r.RoundID, &r.TeamID, &r.Path, &r.Score, &detailsJSON, &r.SubmittedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if detailsJSON.Valid {
		r.Details, _ = model.ScoreDetailsFromJSON(detailsJSON.String)
	}
	return r, nil
}

// GetRoutesByRound returns every team's submitted route for a round.
func (s *SQLiteStore) GetRoutesByRound(roundID string) ([]model.TeamRoute, error) {
	rows, err := s.db.Query("SELECT id, session_id, round_id, team_id, path, score, details, submitted_at FROM team_routes WHERE round_id = ?", roundID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var routes []model.TeamRoute
	for rows.Next() {
		var r model.TeamRoute
		var detailsJSON sql.NullString
		if err := rows.Scan(&r.ID, &r.SessionID, &r.RoundID, &r.TeamID, &r.Path, &r.Score, &detailsJSON, &r.SubmittedAt); err != nil {
			return nil, err
		}
		if detailsJSON.Valid {
			r.Details, _ = model.ScoreDetailsFromJSON(detailsJSON.String)
		}
		routes = append(routes, r)
	}
	return routes, rows.Err()
}

// UpdateTeamRouteScore overwrites the score and details of a stored route.
func (s *SQLiteStore) UpdateTeamRouteScore(id string, score float64, details string) error {
	_, err := s.db.Exec("UPDATE team_routes SET score = ?, details = ? WHERE id = ?", score, details, id)
	return err
}

// DeleteTeamRoute removes a stored team route, allowing resubmission.
func (s *SQLiteStore) DeleteTeamRoute(roundID, teamID string) error {
	_, err := s.db.Exec("DELETE FROM team_routes WHERE round_id = ? AND team_id = ?", roundID, teamID)
	return err
}

// DeletePlayer removes a player row.
func (s *SQLiteStore) DeletePlayer(id string) error {
	_, err := s.db.Exec("DELETE FROM players WHERE id = ?", id)
	return err
}
