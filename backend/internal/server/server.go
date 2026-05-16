package server

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"golang.org/x/time/rate"

	"github.com/colinbradley/sluff/internal/config"
	"github.com/colinbradley/sluff/internal/handler"
	"github.com/colinbradley/sluff/internal/middleware"
	"github.com/colinbradley/sluff/internal/store"
	"github.com/colinbradley/sluff/internal/ws"
)

type Server struct {
	router      *chi.Mux
	store       store.Store
	hub         *ws.Hub
	cfg         *config.Config
	authLimiter *middleware.RateLimiter
}

func New(s store.Store, cfg *config.Config) *Server {
	srv := &Server{
		router:      chi.NewRouter(),
		store:       s,
		hub:         ws.NewHub(),
		cfg:         cfg,
		authLimiter: middleware.NewRateLimiter(rate.Limit(5), 10), // 5 req/sec, burst 10
	}

	go srv.hub.Run()

	srv.setupMiddleware()
	srv.setupRoutes()
	return srv
}

func (s *Server) setupMiddleware() {
	s.router.Use(chimw.Logger)
	s.router.Use(chimw.Recoverer)
	s.router.Use(chimw.RequestID)
	s.router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   s.cfg.CORSOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type", "Authorization"},
		AllowCredentials: true,
		MaxAge:           300,
	}))
}

func (s *Server) setupRoutes() {
	guideH := handler.NewGuideHandler(s.store)
	sessionH := handler.NewSessionHandler(s.store)
	gameH := handler.NewGameHandler(s.store, s.hub)
	wsH := handler.NewWSHandler(s.store, s.hub)
	authH := handler.NewAuthHandler(s.store, s.cfg.JWTSecret)
	adminH := handler.NewGuideAdminHandler(s.store, s.hub)

	guideAuth := middleware.GuideAuth(s.cfg.JWTSecret)

	s.router.Get("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Auth endpoints (public, rate limited)
	s.router.Post("/api/auth/register", func(w http.ResponseWriter, r *http.Request) {
		s.authLimiter.Limit(http.HandlerFunc(authH.Register)).ServeHTTP(w, r)
	})
	s.router.Post("/api/auth/login", func(w http.ResponseWriter, r *http.Request) {
		s.authLimiter.Limit(http.HandlerFunc(authH.Login)).ServeHTTP(w, r)
	})

	// Guide map management (requires auth)
	s.router.Route("/api/guide/maps", func(r chi.Router) {
		r.Use(guideAuth)
		r.Post("/", guideH.CreateMap)
		r.Get("/", guideH.ListMaps)
		r.Get("/{mapID}", guideH.GetMap)
		r.Put("/{mapID}", guideH.UpdateMap)
		r.Delete("/{mapID}", guideH.DeleteMap)
		r.Post("/{mapID}/rounds", guideH.CreateRound)
		r.Put("/{mapID}/rounds/{roundID}", guideH.UpdateRound)
		r.Delete("/{mapID}/rounds/{roundID}", guideH.DeleteRound)
	})

	// Session endpoints
	s.router.Route("/api/sessions", func(r chi.Router) {
		// Guide-only: create sessions
		r.With(guideAuth).Post("/", sessionH.CreateSession)
		r.With(guideAuth).Post("/solo", sessionH.CreateSoloSession)

		// Public: demo session (no auth required)
		r.Post("/demo", sessionH.CreateDemoSession)

		// Public: join/observe
		r.Get("/{sessionID}", sessionH.GetSession)
		r.Get("/code/{code}", sessionH.GetSessionByCode)
		r.Post("/{sessionID}/join", sessionH.JoinSession)
		r.Post("/{sessionID}/teams", sessionH.CreateTeam)
		r.Post("/{sessionID}/teams/{teamID}/join", sessionH.JoinTeam)
		r.Get("/{sessionID}/ws", wsH.HandleWebSocket)

		// Guide-only: game control
		r.With(guideAuth).Post("/{sessionID}/start", gameH.StartGame)
		r.With(guideAuth).Delete("/{sessionID}/players/{playerID}", adminH.KickPlayer)
		r.With(guideAuth).Delete("/{sessionID}/rounds/{roundID}/routes/{teamID}", adminH.ClearRoute)

		// Public: demo round advancement and current round fetch
		r.Post("/{sessionID}/demo/next", gameH.DemoNextRound)
		r.Get("/{sessionID}/current-round", gameH.GetCurrentRound)

		// Scoring (public — players submit their own routes)
		r.Post("/{sessionID}/rounds/{roundID}/submit", gameH.SubmitRoute)
		r.Get("/{sessionID}/rounds/{roundID}/scores", gameH.GetScores)
	})
}

func (s *Server) Start(addr string) error {
	log.Printf("Server starting on %s", addr)
	return http.ListenAndServe(addr, s.router)
}
