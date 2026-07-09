-- Kudos on logs, push-notification device tokens, and a server-owned catalog
-- of every Google place we've surfaced (grows into our own content moat and a
-- fallback when the Places API is unavailable).

CREATE TABLE log_reactions (
    log_id     UUID NOT NULL REFERENCES logs(id) ON DELETE CASCADE,
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (log_id, user_id)
);

CREATE TABLE device_tokens (
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token      TEXT NOT NULL,
    platform   TEXT NOT NULL, -- ANDROID | IOS
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, token)
);

CREATE TABLE places_catalog (
    place_id      TEXT PRIMARY KEY,
    name          TEXT NOT NULL,
    address       TEXT NOT NULL DEFAULT '',
    lat           DOUBLE PRECISION NOT NULL DEFAULT 0,
    lng           DOUBLE PRECISION NOT NULL DEFAULT 0,
    rating        DOUBLE PRECISION NOT NULL DEFAULT 0,
    ratings_total INT NOT NULL DEFAULT 0,
    types         TEXT[] NOT NULL DEFAULT '{}',
    photo_ref     TEXT NOT NULL DEFAULT '',
    first_seen    TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen     TIMESTAMPTZ NOT NULL DEFAULT now()
);
