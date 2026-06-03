package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Nico-Csk/socialflow/internal/domain"
	shttp "github.com/Nico-Csk/socialflow/internal/http"
	"github.com/Nico-Csk/socialflow/internal/service"
	"github.com/Nico-Csk/socialflow/internal/store"
)

func main() {
	cfg := loadConfig()

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("unable to connect to database: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("unable to ping database: %v", err)
	}
	log.Println("connected to database")

	// ----- Store -----
	st := store.NewStore(pool)

	// ----- Services -----
	authSvc := service.NewAuthService(st, pool, []byte(cfg.JWTSecret), cfg.JWTExpiry, cfg.Env)
	wsSvc := service.NewWorkspaceService(st, pool, authSvc)
	clientSvc := service.NewClientService(st, pool)
	contentSvc := service.NewContentService(st, pool)
	commentSvc := service.NewCommentService(st, pool, contentSvc)
	taskSvc := service.NewTaskService(st, pool)
	dashboardSvc := service.NewDashboardService(st, pool)

	// ----- Handlers -----
	authH := shttp.NewAuthHandler(authSvc)
	wsH := shttp.NewWorkspaceHandler(wsSvc, authSvc)
	clientH := shttp.NewClientHandler(clientSvc)
	contentH := shttp.NewContentHandler(contentSvc)
	commentH := shttp.NewCommentHandler(commentSvc)
	taskH := shttp.NewTaskHandler(taskSvc)
	calendarH := shttp.NewCalendarHandler(contentSvc)
	dashboardH := shttp.NewDashboardHandler(dashboardSvc)

	// ----- Router -----
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Health
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	// API v1
	r.Route("/api", func(r chi.Router) {
		// Auth — public (no middleware)
		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", authH.Register)
			r.Post("/login", authH.Login)
			r.Post("/logout", authH.Logout)
		})

		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(shttp.AuthMiddleware(authSvc))

			// Current user
			r.Get("/me", authH.Me)

			// Invite claim — only needs auth, no workspace context
			r.Post("/invites/{token}/claim", wsH.ClaimInvite)

			// Workspaces
			r.Route("/workspaces", func(r chi.Router) {
				r.Get("/", wsH.List)
				r.Post("/", wsH.Create)
				r.Post("/switch", wsH.SwitchActive)

				// Workspace-scoped routes
				r.Route("/{id}", func(r chi.Router) {
					r.Get("/", wsH.Get)
					r.Put("/", wsH.Update)
					r.Delete("/", wsH.Delete)

					// Members
					r.Get("/members", wsH.ListMembers)
					r.Put("/members/{userID}", wsH.UpdateMemberRole)
					r.Delete("/members/{userID}", wsH.RemoveMember)

					// Invites
					r.Post("/invites", wsH.CreateInvite)
				})
			})

			// Workspace-scoped resources (require active workspace)
			r.Group(func(r chi.Router) {
				r.Use(shttp.RequireWorkspace())
				r.Use(shttp.RevalidateWorkspaceMembership(st, pool))

				// Clients — reader+
				r.Route("/clients", func(r chi.Router) {
					r.Get("/", clientH.List)
					r.With(shttp.RequireRole(domain.RoleCM, domain.RoleAdmin)).Post("/", clientH.Create)
					r.Route("/{id}", func(r chi.Router) {
						r.Get("/", clientH.Get)
						r.With(shttp.RequireRole(domain.RoleCM, domain.RoleAdmin)).Put("/", clientH.Update)
						r.With(shttp.RequireRole(domain.RoleCM, domain.RoleAdmin)).Delete("/", clientH.Delete)
					})
				})

				// Content Items — reader+
				r.Route("/content-items", func(r chi.Router) {
					r.Get("/", contentH.List)
					r.With(shttp.RequireRole(domain.RoleCM, domain.RoleAdmin)).Post("/", contentH.Create)
					r.Route("/{id}", func(r chi.Router) {
						r.Get("/", contentH.Get)
						r.With(shttp.RequireRole(domain.RoleCM, domain.RoleAdmin)).Put("/", contentH.Update)
						r.With(shttp.RequireRole(domain.RoleCM, domain.RoleAdmin)).Patch("/status", contentH.TransitionStatus)

						// Comments on content items
						r.Route("/comments", func(r chi.Router) {
							r.Get("/", commentH.List)
							r.With(shttp.RequireRole(domain.RoleCM, domain.RoleAdmin)).Post("/", commentH.Create)
						})
					})
				})

				// Comments — delete by author only (cm+ or author)
				r.With(shttp.RequireRole(domain.RoleCM, domain.RoleAdmin)).Delete("/comments/{commentID}", commentH.Delete)

				// Calendar
				r.Get("/calendar", calendarH.ListByMonth)

				// Dashboard
				r.Get("/dashboard", dashboardH.Summary)

				// Tasks
				r.Route("/tasks", func(r chi.Router) {
					r.Get("/", taskH.List)
					r.With(shttp.RequireRole(domain.RoleCM, domain.RoleAdmin)).Post("/", taskH.Create)
					r.Route("/{id}", func(r chi.Router) {
						r.Get("/", taskH.Get)
						r.With(shttp.RequireRole(domain.RoleCM, domain.RoleAdmin)).Put("/", taskH.Update)
						r.With(shttp.RequireRole(domain.RoleCM, domain.RoleAdmin)).Delete("/", taskH.Delete)
					})
				})
			})
		})
	})

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("shutting down gracefully...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Printf("shutdown error: %v", err)
		}
	}()

	log.Printf("SocialFlow API listening on :%s", cfg.Port)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
	log.Println("server stopped")
}

type config struct {
	Port        string
	DatabaseURL string
	JWTSecret   string
	JWTExpiry   time.Duration
	Env         string
}

func loadConfig() config {
	port := envOrDefault("PORT", "8080")
	dbURL := envOrDefault("DATABASE_URL", "postgres://socialflow:socialflow@localhost:5432/socialflow?sslmode=disable")
	jwtSecret := envOrDefault("JWT_SECRET", "dev-secret-change-me")
	jwtExpiryStr := envOrDefault("JWT_EXPIRY_HOURS", "72")

	jwtExpiry, err := time.ParseDuration(jwtExpiryStr + "h")
	if err != nil {
		jwtExpiry = 72 * time.Hour
	}

	return config{
		Port:        port,
		DatabaseURL: dbURL,
		JWTSecret:   jwtSecret,
		JWTExpiry:   jwtExpiry,
		Env:         envOrDefault("ENV", "development"),
	}
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
