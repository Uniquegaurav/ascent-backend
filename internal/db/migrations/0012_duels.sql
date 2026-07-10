-- Head-to-head duels (design ref 2): two users compete on logged activity over
-- a fixed window. Progress is derived live from the logs table, so there's no
-- counter to keep in sync — the row just records who, what, and when.
CREATE TABLE duels (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    challenger  UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    opponent    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    -- Optional interest filter; NULL means "any activity".
    interest_id TEXT,
    starts_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    ends_at     TIMESTAMPTZ NOT NULL,
    -- PENDING (awaiting opponent), ACTIVE (accepted), DECLINED.
    status      TEXT NOT NULL DEFAULT 'PENDING',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX duels_challenger_idx ON duels (challenger);
CREATE INDEX duels_opponent_idx ON duels (opponent);
