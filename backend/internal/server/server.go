// Package server wires the chi router, middleware, and HTTP handlers and
// exposes a Server type that owns the http.Server lifecycle.
package server

import (
	"context"
	"log/slog"
	"net/http"
	"time"

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

// Server is the HTTP application: chi router, middleware stack, hub, and store.
type Server struct {
	router      *chi.Mux
	store       *store.SQLiteStore
	hub         *ws.Hub
	cfg         *config.Config
	authLimiter *middleware.RateLimiter
}

// New constructs a Server, starts the WebSocket hub, and registers all routes.
// The rate limiter's cleanup goroutine is tied to ctx so it stops when the
// caller cancels.
func New(ctx context.Context, s *store.SQLiteStore, cfg *config.Config) *Server {
	srv := &Server{
		router:      chi.NewRouter(),
		store:       s,
		hub:         ws.NewHub(),
		cfg:         cfg,
		authLimiter: middleware.NewRateLimiter(ctx, rate.Limit(5), 10), // 5 req/sec, burst 10
	}

	go srv.hub.Run()

	srv.setupMiddleware()
	srv.setupRoutes(ctx)
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
		AllowCredentials: false,
		MaxAge:           300,
	}))
	s.router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 8 MB accommodates large multi-round GeoJSON/KML imports with
			// many corridor + no-go-zone vertices.
			r.Body = http.MaxBytesReader(w, r.Body, 8<<20)
			next.ServeHTTP(w, r)
		})
	})
}

func (s *Server) setupRoutes(ctx context.Context) {
	guideH := handler.NewGuideHandler(s.store)
	sessionH := handler.NewSessionHandler(s.store)
	gameH := handler.NewGameHandler(s.store, s.hub)
	wsH := handler.NewWSHandler(s.store, s.hub)
	authH := handler.NewAuthHandler(s.store, s.cfg.JWTSecret)
	adminH := handler.NewGuideAdminHandler(s.store, s.hub)

	go gameH.RunRoundTicker(ctx)

	guideAuth := middleware.GuideAuth(s.cfg.JWTSecret)

	s.router.Get("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	// Auth endpoints (public, rate limited)
	s.router.With(s.authLimiter.Limit).Post("/api/auth/register", authH.Register)
	s.router.With(s.authLimiter.Limit).Post("/api/auth/login", authH.Login)

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
		r.With(guideAuth).Post("/{sessionID}/end-round", gameH.EndRound)
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

// Start runs the HTTP server with sane timeouts. It blocks until the server
// stops or returns an error.
//
// ReadTimeout and WriteTimeout are intentionally omitted: the same router
// serves long-lived WebSocket upgrades, and a Server-wide write deadline
// persists on the hijacked TCP connection and would close WebSockets after
// the timeout regardless of activity. ReadHeaderTimeout still guards against
// slowloris on the header read, and IdleTimeout reaps idle keep-alives.
func (s *Server) Start(addr string) error {
	httpSrv := &http.Server{
		Addr:              addr,
		Handler:           s.router,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	slog.Info("server starting", "addr", addr)
	return httpSrv.ListenAndServe()
}
