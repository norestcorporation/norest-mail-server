package httpserver

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/norest-mail/server/internal/addresses"
	"github.com/norest-mail/server/internal/auth"
	"github.com/norest-mail/server/internal/billing"
	"github.com/norest-mail/server/internal/config"
	"github.com/norest-mail/server/internal/db"
	"github.com/norest-mail/server/internal/domains"
	"github.com/norest-mail/server/internal/mail"
	"github.com/norest-mail/server/internal/metrics"
	"github.com/norest-mail/server/internal/policy"
	"github.com/norest-mail/server/internal/ratelimit"
	"github.com/norest-mail/server/internal/response"
	"github.com/norest-mail/server/internal/stalwart"
)

// NewRouter creates the application router with all routes mounted.
func NewRouter(cfg *config.Config, pool *pgxpool.Pool, stalwartClient *stalwart.Client) http.Handler {
	r := chi.NewRouter()

	// Rate limiters
	loginLimiter := ratelimit.NewLimiter(&ratelimit.Config{
		Requests: 10,
		Window:   time.Minute,
	})
	registerLimiter := ratelimit.NewLimiter(&ratelimit.Config{
		Requests: 5,
		Window:   time.Hour,
	})
	mailSessionLimiter := ratelimit.NewLimiter(&ratelimit.Config{
		Requests: 10,
		Window:   time.Minute,
	})
	adminLimiter := ratelimit.NewLimiter(&ratelimit.Config{
		Requests: 5,
		Window:   time.Minute,
	})

	// Middleware
	r.Use(Recoverer)
	r.Use(RequestID)
	r.Use(RequestLogger)
	r.Use(SecurityHeaders)
	r.Use(CORSMiddleware(cfg))
	r.Use(RequestSizeLimit(1 << 20)) // 1MB max request body
	r.Use(BackpressureMiddleware(pool))

	// Health endpoints
	r.Get("/health", handleHealth())
	r.Get("/health/live", handleHealthLive())
	r.Get("/health/ready", handleHealthReady(pool, stalwartClient))
	r.Get("/health/db", handleHealthDB(pool))
	r.Get("/health/stalwart", handleHealthStalwart(stalwartClient))
	r.Get("/metrics", handleMetrics())

	// Services
	authService := auth.NewService(pool, cfg.JWTSecret)
	authHandler := auth.NewHandler(authService)

	domainsService := domains.NewService(pool)
	domainsHandler := domains.NewHandler(domainsService)

	addressesService := addresses.NewService(pool)
	addressesHandler := addresses.NewHandler(addressesService)

	billingService := billing.NewService(pool, nil)
	billingHandler := billing.NewHandler(billingService)

	policyService := policy.NewService(pool)
	policyHandler := policy.NewHandler(policyService)

	dbImpl := db.NewMailRepository(pool)
	// We assume Stalwart is accessible at localhost:8081 for the JMAP frontend (per Ch3 dev URL rules)
	mailService := mail.NewService(dbImpl, stalwartClient)
	mailHandler := mail.NewHandler(mailService, "http://localhost:8081")

	// API v1 — prepared for Chapter 2
	r.Route("/v1", func(r chi.Router) {
		r.Route("/auth", func(r chi.Router) {
			r.With(registerLimiter.Middleware(ratelimit.IPKey)).Post("/register", authHandler.Register)
			r.With(loginLimiter.Middleware(ratelimit.IPKey)).Post("/login", authHandler.Login)
		})

		r.With(auth.RequireAuth(authService)).Get("/me", authHandler.Me)

		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(auth.RequireAuth(authService))

			// Domains
			r.Route("/domains", func(r chi.Router) {
				r.Post("/", domainsHandler.Create)
				r.Get("/", domainsHandler.List)
				r.Get("/{id}", domainsHandler.Get)
				r.Delete("/{id}", domainsHandler.Delete)
				r.Post("/{id}/verification/start", domainsHandler.StartVerification)
				r.Get("/{id}/verification", domainsHandler.GetVerification)

				// Addresses
				r.Route("/{domainID}/addresses", func(r chi.Router) {
					r.Post("/", addressesHandler.Create)
					r.Get("/", addressesHandler.List)
				})
			})

			// Mail Layer
			r.Route("/mail", func(r chi.Router) {
				r.With(mailSessionLimiter.Middleware(ratelimit.UserKey)).Post("/session", mailHandler.CreateSession)
				r.Get("/account", mailHandler.GetAccount)
			})

			// Product Layer
			r.Route("/account", func(r chi.Router) {
				r.Get("/usage", policyHandler.GetUsage)
			})

			// Admin Layer
			r.Group(func(r chi.Router) {
				r.Use(auth.RequireAdmin(authService))
				r.Use(adminLimiter.Middleware(ratelimit.UserKey))
				r.Route("/admin", func(r chi.Router) {
					r.Post("/accounts/{id}/suspend", policyHandler.SuspendAccount)
					r.Post("/accounts/{id}/reactivate", policyHandler.ReactivateAccount)
				})
			})

		})

		// Billing Layer (External webhooks, not protected by user auth)
		r.Route("/billing", func(r chi.Router) {
			r.Use(RequestSizeLimit(100 << 10)) // 100KB for webhooks
			r.Post("/webhook", billingHandler.HandleWebhook)
		})
	})

	return r
}

func handleHealth() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		response.OK(w, map[string]string{"status": "ok"})
	}
}

func handleHealthLive() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Liveness: process is alive
		response.OK(w, map[string]string{"status": "alive"})
	}
}

func handleHealthReady(pool *pgxpool.Pool, stalwartClient *stalwart.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		
		// Readiness: can serve expected traffic
		// Check database
		if err := db.HealthCheck(ctx, pool); err != nil {
			response.ServiceError(w, http.StatusServiceUnavailable, "not_ready", "database unhealthy")
			return
		}
		
		// Check Stalwart with timeout
		healthCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		
		if err := stalwartClient.HealthCheck(healthCtx); err != nil {
			response.ServiceError(w, http.StatusServiceUnavailable, "not_ready", "stalwart unhealthy")
			return
		}
		
		response.OK(w, map[string]string{"status": "ready"})
	}
}

func handleHealthDB(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := db.HealthCheck(r.Context(), pool); err != nil {
			response.ServiceError(w, http.StatusServiceUnavailable, "database", err.Error())
			return
		}
		response.OK(w, map[string]string{"status": "ok", "service": "database"})
	}
}

func handleHealthStalwart(client *stalwart.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Use a short timeout for health checks
		healthCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		if err := client.HealthCheck(healthCtx); err != nil {
			response.ServiceError(w, http.StatusServiceUnavailable, "stalwart", err.Error())
			return
		}
		response.OK(w, map[string]string{"status": "ok", "service": "stalwart"})
	}
}

func handleMetrics() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		response.OK(w, metrics.GetSnapshot())
	}
}
