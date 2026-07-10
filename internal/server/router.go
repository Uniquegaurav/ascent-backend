package server

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kumargaurav/summit-backend/internal/ascent"
	"github.com/kumargaurav/summit-backend/internal/auth"
	"github.com/kumargaurav/summit-backend/internal/catalog"
	"github.com/kumargaurav/summit-backend/internal/challenge"
	"github.com/kumargaurav/summit-backend/internal/config"
	"github.com/kumargaurav/summit-backend/internal/discovery"
	"github.com/kumargaurav/summit-backend/internal/explore"
	"github.com/kumargaurav/summit-backend/internal/friend"
	"github.com/kumargaurav/summit-backend/internal/hobby"
	"github.com/kumargaurav/summit-backend/internal/httpx"
	"github.com/kumargaurav/summit-backend/internal/integration"
	"github.com/kumargaurav/summit-backend/internal/places"
	"github.com/kumargaurav/summit-backend/internal/providers"
	"github.com/kumargaurav/summit-backend/internal/user"
	"github.com/kumargaurav/summit-backend/internal/wishlist"
)

func New(pool *pgxpool.Pool, cfg config.Config) http.Handler {
	authRepo := auth.NewRepo(pool)
	tokens := auth.NewTokenManager(cfg.JWTSecret, cfg.AccessTokenTTL)
	authSvc := auth.NewService(authRepo, auth.NewSender(cfg), tokens, auth.ServiceOpts{
		DevCode:     cfg.OTPDevCode,
		ReviewPhone: cfg.OTPReviewPhone,
		Dev:         cfg.IsDev(),
		RefreshTTL:  cfg.RefreshTokenTTL,
	})
	authH := auth.NewHandler(authSvc)

	pc := places.New(cfg.GooglePlacesKey)
	prov := providers.New()
	userH := user.NewHandler(user.NewRepo(pool))
	exploreH := explore.NewHandler(pc, pool, prov)
	ascentH := ascent.NewHandler(ascent.NewRepo(pool))
	friendH := friend.NewHandler(friend.NewRepo(pool))
	discoveryH := discovery.NewHandler(discovery.NewService(pc))
	catalogH := catalog.NewHandler(catalog.NewRepo(pool))
	challengeH := challenge.NewHandler(challenge.NewRepo(pool))
	hobbyH := hobby.NewHandler(hobby.NewRepo(pool))
	integrationH := integration.NewHandler(integration.NewRepo(pool))
	wishlistH := wishlist.NewHandler(wishlist.NewRepo(pool))

	r := chi.NewRouter()
	r.Use(httpx.Recover, httpx.RequestID, httpx.RequestLogger, httpx.CORS,
		httpx.RateLimit(httpx.NewRateLimiter(300, time.Minute)))

	r.Get("/health", func(w http.ResponseWriter, req *http.Request) {
		ctx, cancel := context.WithTimeout(req.Context(), 2*time.Second)
		defer cancel()
		if err := pool.Ping(ctx); err != nil {
			httpx.Error(w, http.StatusServiceUnavailable, "database unreachable")
			return
		}
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	// Public: image proxy + avatar serving (image loaders can't send the JWT).
	r.Get("/place-photo", exploreH.Photo)
	r.Get("/avatars/{id}", userH.ServeAvatar)

	// Auth endpoints get a tighter per-IP window on top of the per-phone
	// limits inside the service (OTP sends cost real money once SMS is live).
	r.Group(func(ar chi.Router) {
		ar.Use(httpx.RateLimit(httpx.NewRateLimiter(30, time.Hour)))
		ar.Post("/auth/request-otp", authH.RequestOTP)
		ar.Post("/auth/verify-otp", authH.VerifyOTP)
	})
	r.Post("/auth/refresh", authH.Refresh)

	// Guest-browsable: discovery content works logged-out (auth still resolves
	// when a token is sent, so trending stays personalized for members).
	r.Group(func(gr chi.Router) {
		gr.Use(httpx.OptionalAuth(tokens.Validate))

		gr.Get("/catalog/interests", catalogH.Interests)
		gr.Get("/catalog/cities", catalogH.Cities)
		gr.Get("/catalog/searches", catalogH.PopularSearches)
		gr.Get("/catalog/categories", catalogH.Categories)

		gr.Get("/explore", exploreH.Feed)
		gr.Get("/explore/search", exploreH.Search)
		gr.Get("/explore/{id}", exploreH.Detail)
		gr.Get("/treks", exploreH.Treks)
		gr.Get("/geocode/reverse", exploreH.ReverseGeocode)
	})

	r.Group(func(pr chi.Router) {
		pr.Use(httpx.RequireAuth(tokens.Validate))

		pr.Get("/users/me", userH.Me)
		pr.Delete("/users/me", userH.DeleteMe)
		pr.Post("/users/onboarding", userH.Onboarding)
		pr.Post("/users/name", userH.UpdateName)
		pr.Get("/users/hobbies", userH.Hobbies)
		pr.Put("/users/hobbies", userH.SetHobbies)
		pr.Post("/devices", userH.RegisterDevice)
		pr.Get("/hobbies/{id}/guide", hobbyH.Guide)
		pr.Get("/hobbies/{id}/picks", hobbyH.Picks)
		pr.Get("/hobbies/discover", hobbyH.Discover)

		pr.Get("/integrations", integrationH.List)
		pr.Post("/integrations/{id}/connect", integrationH.Connect)
		pr.Post("/integrations/{id}/disconnect", integrationH.Disconnect)
		pr.Post("/users/avatar", userH.SetAvatar)

		pr.Get("/ascents", ascentH.List)
		pr.Post("/ascents", ascentH.Create)
		pr.Get("/ascents/{id}", ascentH.Detail)
		pr.Post("/ascents/{id}/logs", ascentH.AddLog)
		pr.Post("/ascents/{id}/summit", ascentH.Summit)
		pr.Get("/logs", ascentH.AllLogs)

		pr.Get("/wishlist", wishlistH.List)
		pr.Post("/wishlist", wishlistH.Add)
		pr.Put("/wishlist/{id}", wishlistH.UpdatePlan)
		pr.Delete("/wishlist/{id}", wishlistH.Remove)

		pr.Get("/feed", ascentH.Feed)
		pr.Get("/experiences/feed", ascentH.Feed)
		pr.Post("/logs/{id}/reactions", ascentH.React)
		pr.Delete("/logs/{id}/reactions", ascentH.Unreact)

		pr.Get("/challenges", challengeH.List)
		pr.Post("/challenges/{id}/join", challengeH.Join)
		pr.Post("/challenges/{id}/leave", challengeH.Leave)

		pr.Get("/friends/list", friendH.List)
		pr.Post("/friends/sync-contacts", friendH.SyncContacts)
		pr.Post("/friends/{id}/request", friendH.Request)
		pr.Post("/friends/{id}/accept", friendH.Accept)
		pr.Post("/friends/{id}/invite", ascentH.Invite)
		pr.Get("/friends/{id}/logs", ascentH.FriendLogs)


		pr.Get("/places", discoveryH.Places)
		pr.Get("/events", discoveryH.Events)
	})

	return r
}
