package store

import "github.com/colinbradley/sluff/internal/model"

// Store defines all data access operations.
type Store interface {
	// Maps
	CreateMap(m *model.GameMap) error
	GetMap(id string) (*model.GameMap, error)
	ListMaps() ([]model.GameMap, error)
	UpdateMap(m *model.GameMap) error
	DeleteMap(id string) error

	// Rounds
	CreateRound(r *model.Round) error
	GetRound(id string) (*model.Round, error)
	GetRoundsByMap(mapID string) ([]model.Round, error)
	UpdateRound(r *model.Round) error
	DeleteRound(id string) error

	// Sessions
	CreateSession(s *model.Session) error
	GetSession(id string) (*model.Session, error)
	GetSessionByCode(code string) (*model.Session, error)
	UpdateSession(s *model.Session) error

	// Teams
	CreateTeam(t *model.Team) error
	GetTeamsBySession(sessionID string) ([]model.Team, error)

	// Players
	CreatePlayer(p *model.Player) error
	GetPlayer(id string) (*model.Player, error)
	GetPlayersBySession(sessionID string) ([]model.Player, error)
	UpdatePlayerTeam(playerID, teamID string) error

	// Routes
	CreateTeamRoute(r *model.TeamRoute) error
	GetTeamRoute(roundID, teamID string) (*model.TeamRoute, error)
	GetRoutesByRound(roundID string) ([]model.TeamRoute, error)
	UpdateTeamRouteScore(id string, score float64, details string) error

	Close() error
}
