package server

import (
	"net/http"

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
	"github.com/kumargaurav/summit-backend/internal/httpx"
	"github.com/kumargaurav/summit-backend/internal/places"
	"github.com/kumargaurav/summit-backend/internal/user"
)

func New(pool *pgxpool.Pool, cfg config.Config) http.Handler {
	authRepo := auth.NewRepo(pool)
	tokens := auth.NewTokenManager(cfg.JWTSecret, cfg.AccessTokenTTL)
	authSvc := auth.NewService(authRepo, auth.LogSender{}, tokens, cfg.OTPDevCode, cfg.IsDev(), cfg.RefreshTokenTTL)
	authH := auth.NewHandler(authSvc)

	pc := places.New(cfg.GooglePlacesKey)
	userH := user.NewHandler(user.NewRepo(pool))
	exploreH := explore.NewHandler(pc)
	ascentH := ascent.NewHandler(ascent.NewRepo(pool))
	friendH := friend.NewHandler(friend.NewRepo(pool))
	challengeH := challenge.NewHandler(challenge.NewRepo(pool))
	discoveryH := discovery.NewHandler(discovery.NewService(pc))
	catalogH := catalog.NewHandler(catalog.NewRepo(pool))

	r := chi.NewRouter()
	r.Use(httpx.Recover, httpx.RequestLogger, httpx.CORS)

	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	// Public: image proxy (Coil can't send the JWT).
	r.Get("/place-photo", exploreH.Photo)
	// Public temporary diagnostic for the Google key/API setup.
	r.Get("/debug/places", exploreH.Debug)

	r.Post("/auth/request-otp", authH.RequestOTP)
	r.Post("/auth/verify-otp", authH.VerifyOTP)
	r.Post("/auth/refresh", authH.Refresh)

	r.Group(func(pr chi.Router) {
		pr.Use(httpx.RequireAuth(tokens.Validate))

		pr.Get("/users/me", userH.Me)
		pr.Post("/users/onboarding", userH.Onboarding)
		pr.Post("/users/name", userH.UpdateName)
		pr.Get("/users/hobbies", userH.Hobbies)
		pr.Put("/users/hobbies", userH.SetHobbies)

		pr.Get("/catalog/interests", catalogH.Interests)
		pr.Get("/catalog/cities", catalogH.Cities)
		pr.Get("/catalog/searches", catalogH.PopularSearches)
		pr.Get("/catalog/categories", catalogH.Categories)

		pr.Get("/explore", exploreH.Feed)
		pr.Get("/explore/search", exploreH.Search)
		pr.Get("/explore/{id}", exploreH.Detail)
		pr.Get("/geocode/reverse", exploreH.ReverseGeocode)

		pr.Get("/ascents", ascentH.List)
		pr.Post("/ascents", ascentH.Create)
		pr.Get("/ascents/{id}", ascentH.Detail)
		pr.Post("/ascents/{id}/logs", ascentH.AddLog)
		pr.Post("/ascents/{id}/summit", ascentH.Summit)
		pr.Get("/logs", ascentH.AllLogs)

		pr.Get("/feed", ascentH.Feed)
		pr.Get("/experiences/feed", ascentH.Feed)

		pr.Get("/friends/list", friendH.List)
		pr.Post("/friends/sync-contacts", friendH.SyncContacts)
		pr.Post("/friends/{id}/request", friendH.Request)
		pr.Post("/friends/{id}/accept", friendH.Accept)
		pr.Post("/friends/{id}/invite", ascentH.Invite)
		pr.Get("/friends/{id}/logs", ascentH.FriendLogs)

		pr.Get("/challenges", challengeH.List)
		pr.Post("/challenges/{id}/join", challengeH.Join)
		pr.Post("/challenges/{id}/complete", challengeH.Complete)

		pr.Get("/places", discoveryH.Places)
		pr.Get("/events", discoveryH.Events)
	})

	return r
}
