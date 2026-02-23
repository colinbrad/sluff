package store

const schema = `
CREATE TABLE IF NOT EXISTS game_maps (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	description TEXT NOT NULL DEFAULT '',
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS rounds (
	id TEXT PRIMARY KEY,
	map_id TEXT NOT NULL REFERENCES game_maps(id) ON DELETE CASCADE,
	round_number INTEGER NOT NULL,
	name TEXT NOT NULL DEFAULT '',
	start_point TEXT NOT NULL,
	end_point TEXT NOT NULL,
	corridor TEXT NOT NULL,
	UNIQUE(map_id, round_number)
);

CREATE TABLE IF NOT EXISTS sessions (
	id TEXT PRIMARY KEY,
	map_id TEXT NOT NULL REFERENCES game_maps(id),
	code TEXT UNIQUE NOT NULL,
	phase TEXT NOT NULL DEFAULT 'waiting',
	current_round INTEGER NOT NULL DEFAULT 0,
	time_limit_sec INTEGER NOT NULL DEFAULT 300,
	is_solo INTEGER NOT NULL DEFAULT 0,
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS teams (
	id TEXT PRIMARY KEY,
	session_id TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
	name TEXT NOT NULL,
	color TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS players (
	id TEXT PRIMARY KEY,
	session_id TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
	team_id TEXT REFERENCES teams(id),
	name TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS team_routes (
	id TEXT PRIMARY KEY,
	session_id TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
	round_id TEXT NOT NULL REFERENCES rounds(id) ON DELETE CASCADE,
	team_id TEXT NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
	path TEXT NOT NULL,
	score REAL,
	details TEXT,
	submitted_at DATETIME,
	UNIQUE(round_id, team_id)
);

CREATE INDEX IF NOT EXISTS idx_rounds_map_id ON rounds(map_id);
CREATE INDEX IF NOT EXISTS idx_sessions_code ON sessions(code);
CREATE INDEX IF NOT EXISTS idx_players_session_id ON players(session_id);
CREATE INDEX IF NOT EXISTS idx_teams_session_id ON teams(session_id);
CREATE INDEX IF NOT EXISTS idx_team_routes_round_id ON team_routes(round_id);
`

// alterMigrations run after schema creation to add columns to existing tables.
// Errors from "duplicate column name" are expected and ignored.
var alterMigrations = []string{
	"ALTER TABLE sessions ADD COLUMN is_solo INTEGER NOT NULL DEFAULT 0",
}
