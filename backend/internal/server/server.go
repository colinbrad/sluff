package server

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/colinbradley/sluff/internal/config"
	"github.com/colinbradley/sluff/internal/handler"
	"github.com/colinbradley/sluff/internal/store"
	"github.com/colinbradley/sluff/internal/ws"
)

type Server struct {
	router *chi.Mux
	store  store.Store
	hub    *ws.Hub
	cfg    *config.Config
}

func New(s store.Store, cfg *config.Config) *Server {
	srv := &Server{
		router: chi.NewRouter(),
		store:  s,
		hub:    ws.NewHub(),
		cfg:    cfg,
	}

	go srv.hub.Run()

	srv.setupMiddleware()
	srv.setupRoutes()
	return srv
}

func (s *Server) setupMiddleware() {
	s.router.Use(middleware.Logger)
	s.router.Use(middleware.Recoverer)
	s.router.Use(middleware.RequestID)
	s.router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   s.cfg.CORSOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))
}

func (s *Server) setupRoutes() {
	adminH := handler.NewAdminHandler(s.store)
	sessionH := handler.NewSessionHandler(s.store)
	gameH := handler.NewGameHandler(s.store, s.hub)
	wsH := handler.NewWSHandler(s.store, s.hub)

	s.router.Get("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Admin endpoints
	s.router.Route("/api/admin/maps", func(r chi.Router) {
		r.Post("/", adminH.CreateMap)
		r.Get("/", adminH.ListMaps)
		r.Get("/{mapID}", adminH.GetMap)
		r.Put("/{mapID}", adminH.UpdateMap)
		r.Delete("/{mapID}", adminH.DeleteMap)
		r.Post("/{mapID}/rounds", adminH.CreateRound)
		r.Put("/{mapID}/rounds/{roundID}", adminH.UpdateRound)
		r.Delete("/{mapID}/rounds/{roundID}", adminH.DeleteRound)
	})

	// Player-facing map listing
	s.router.Get("/api/maps", adminH.ListMaps)

	// Session endpoints
	s.router.Route("/api/sessions", func(r chi.Router) {
		r.Post("/", sessionH.CreateSession)
		r.Post("/solo", sessionH.CreateSoloSession)
		r.Get("/{sessionID}", sessionH.GetSession)
		r.Get("/code/{code}", sessionH.GetSessionByCode)
		r.Post("/{sessionID}/join", sessionH.JoinSession)
		r.Post("/{sessionID}/teams", sessionH.CreateTeam)
		r.Post("/{sessionID}/teams/{teamID}/join", sessionH.JoinTeam)
		r.Post("/{sessionID}/start", gameH.StartGame)
		r.Post("/{sessionID}/rounds/{roundID}/submit", gameH.SubmitRoute)
		r.Get("/{sessionID}/rounds/{roundID}/scores", gameH.GetScores)
		r.Get("/{sessionID}/ws", wsH.HandleWebSocket)
	})
}

func (s *Server) Start(addr string) error {
	log.Printf("Server starting on %s", addr)
	return http.ListenAndServe(addr, s.router)
}
