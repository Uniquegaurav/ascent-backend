-- My-Ascent hierarchy: an ascent can sit under a hobby parent and carry its interest.
ALTER TABLE ascents ADD COLUMN IF NOT EXISTS parent_id   UUID REFERENCES ascents(id) ON DELETE SET NULL;
ALTER TABLE ascents ADD COLUMN IF NOT EXISTS interest_id TEXT;

CREATE INDEX IF NOT EXISTS ascents_parent_idx ON ascents (parent_id);

-- Wishlist: Explore places the climber wants to visit (planned, never logged).
-- The full ExploreItem is snapshotted as JSONB so the card renders without a re-fetch.
CREATE TABLE IF NOT EXISTS wishlists (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id            UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    explore_item_id    TEXT NOT NULL,
    item               JSONB NOT NULL,
    planned_date       TIMESTAMPTZ,
    booking_url        TEXT NOT NULL DEFAULT '',
    added_to_calendar  BOOLEAN NOT NULL DEFAULT FALSE,
    invited_friend_ids TEXT[] NOT NULL DEFAULT '{}',
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS wishlists_user_item_idx ON wishlists (user_id, explore_item_id);
CREATE INDEX IF NOT EXISTS wishlists_user_idx ON wishlists (user_id, created_at DESC);
