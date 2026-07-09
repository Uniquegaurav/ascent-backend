-- Reset all user-generated content so the climber starts fresh.
-- Keeps accounts/auth (users, refresh_tokens), the catalog (interests, cities, challenges,
-- popular_searches) and avatars. Wipes hobby/interest picks, ascents, logs, and the wishlist.
--
-- Run against the database, e.g.:
--   psql "$DATABASE_URL" -f scripts/reset-user-data.sql
-- (locally: `make db-up` then `psql postgres://summit:summit@localhost:5432/summit -f scripts/reset-user-data.sql`)

BEGIN;

TRUNCATE TABLE
    logs,
    ascent_invites,
    ascents,
    wishlists,
    user_hobbies,
    user_interests,
    user_challenges
RESTART IDENTITY CASCADE;

COMMIT;
