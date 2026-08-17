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
	"github.com/norest-mail/server/internal/realtime"
	"github.com/norest-mail/server/internal/registration"
	"github.com/norest-mail/server/internal/response"
	"github.com/norest-mail/server/internal/stalwart"
)

// NewRouter creates the application router with all routes mounted.
func NewRouter(cfg *config.Config, pool *pgxpool.Pool, stalwartClient *stalwart.Client, wsBroker *realtime.Broker) http.Handler {
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

	policyService := policy.NewService(pool)
	policyHandler := policy.NewHandler(policyService)

	registrationEnhancedService := registration.NewEnhancedService(pool, authService, domainsService, addressesService, policyService)
	registrationEnhancedHandler := registration.NewEnhancedHandler(registrationEnhancedService)

	registrationHandler := registration.NewHandler(authService, domainsService, addressesService, pool)

	billingService := billing.NewService(pool, nil)
	billingHandler := billing.NewHandler(billingService)

	dbImpl := db.NewMailRepository(pool)
	idemImpl := db.NewIdempotencyRepository(pool)
	// We assume Stalwart is accessible at localhost:8081 for the JMAP frontend (per Ch3 dev URL rules)
	mailService := mail.NewService(dbImpl, stalwartClient, pool)
	mailHandler := mail.NewHandler(mailService, idemImpl, "http://localhost:8081")

	// API v1 — prepared for Chapter 2
	r.Route("/v1", func(r chi.Router) {
		r.Route("/auth", func(r chi.Router) {
			r.With(registerLimiter.Middleware(ratelimit.IPKey)).Post("/register", registrationEnhancedHandler.Register)
			r.With(loginLimiter.Middleware(ratelimit.IPKey)).Post("/login", authHandler.Login)
		})

		r.With(auth.RequireAuth(authService)).Get("/me", authHandler.Me)

		// Registration flow endpoints
		r.Route("/registration", func(r chi.Router) {
			r.With(auth.RequireAuth(authService)).Get("/status", registrationHandler.GetRegistrationStatus)
			r.With(auth.RequireAuth(authService)).Post("/domains/{domainID}/verify", registrationHandler.InitiateDomainVerification)
			r.With(auth.RequireAuth(authService)).Get("/domains/{domainID}/verify", registrationHandler.CheckDomainVerificationStatus)
		})

		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(auth.RequireAuth(authService))

			// Domains
			r.Route("/domains", func(r chi.Router) {
				r.Get("/platform", domainsHandler.ListPlatformDomains)
				r.Get("/check/{name}", domainsHandler.CheckDomainByName)
				r.Post("/", domainsHandler.Create)
				r.Get("/", domainsHandler.List)
				r.Post("/{id}/verification/start", domainsHandler.StartVerification)
				r.Get("/{id}/verification", domainsHandler.GetVerification)
				r.Get("/{id}", domainsHandler.Get)
				r.Delete("/{id}", domainsHandler.Delete)

				// Addresses
				r.Route("/{domainID}/addresses", func(r chi.Router) {
					r.Post("/reserve", addressesHandler.Reserve)
					r.Get("/", addressesHandler.List)
					r.Get("/check/{localPart}", addressesHandler.CheckAvailability)
				})

				// Address operations
				r.Route("/addresses", func(r chi.Router) {
					r.Post("/{addressID}/claim", addressesHandler.Claim)
				})
			})

			// Mail Layer
			r.Route("/mail", func(r chi.Router) {
				// Existing
				r.With(mailSessionLimiter.Middleware(ratelimit.UserKey)).Post("/session", mailHandler.CreateSession)
				r.Get("/account", mailHandler.GetAccount)
				r.Get("/provisioning-status", mailHandler.GetProvisioningStatus)

				// Realtime
				r.Get("/realtime", realtime.Handler(wsBroker))

				// Attachments
				r.With(RequestSizeLimit(25<<20)).Post("/attachments", mailHandler.UploadAttachment)
				r.Get("/attachments/{blob_id}", mailHandler.DownloadAttachment)

				// Threads
				r.Get("/threads", mailHandler.ListThreads)
				r.Get("/threads/{id}", mailHandler.GetThread)
				r.Get("/threads/{id}/messages", mailHandler.GetThreadMessages)

				// Mailboxes
				r.Get("/mailboxes", mailHandler.ListMailboxes)
				r.Get("/mailboxes/{id}", mailHandler.GetMailbox)

				// Messages
				r.Get("/search", mailHandler.SearchMessages)
				r.Get("/messages", mailHandler.ListMessages)
				r.Get("/messages/{id}", mailHandler.GetMessage)

				// Message Actions
				r.Post("/messages/{id}/read", mailHandler.MarkRead)
				r.Post("/messages/{id}/unread", mailHandler.MarkUnread)
				r.Post("/messages/{id}/star", mailHandler.StarMessage)
				r.Post("/messages/{id}/unstar", mailHandler.UnstarMessage)
				r.Post("/messages/{id}/archive", mailHandler.ArchiveMessage)
				r.Post("/messages/{id}/move", mailHandler.MoveMessage)
				r.Post("/messages/{id}/trash", mailHandler.TrashMessage)
				r.Post("/messages/{id}/restore", mailHandler.RestoreMessage)
				r.Post("/messages/{id}/spam", mailHandler.SpamMessage)

				// Replies and Forwarding
				r.Post("/messages/{id}/reply", mailHandler.ReplyMessage)
				r.Post("/messages/{id}/reply-all", mailHandler.ReplyAllMessage)
				r.Post("/messages/{id}/forward", mailHandler.ForwardMessage)

				// Drafts
				r.Post("/drafts", mailHandler.CreateDraft)
				r.Get("/drafts/{id}", mailHandler.GetDraft)
				r.Put("/drafts/{id}", mailHandler.UpdateDraft)
				r.Delete("/drafts/{id}", mailHandler.DeleteDraft)

				// Sync
				r.Post("/sync", mailHandler.SyncMail)

				// Send
				r.Post("/send", mailHandler.SendMessage)
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
