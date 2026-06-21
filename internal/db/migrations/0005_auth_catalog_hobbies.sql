-- Refresh tokens: opaque, hashed, rotated on use.
CREATE TABLE refresh_tokens (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash  TEXT NOT NULL UNIQUE,
    expires_at  TIMESTAMPTZ NOT NULL,
    revoked     BOOLEAN NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX refresh_tokens_user_idx ON refresh_tokens (user_id);

-- The user's customised home hobbies: ordered selection + optional per-hobby theme.
CREATE TABLE user_hobbies (
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    interest_id TEXT NOT NULL REFERENCES interests(id),
    position    INT  NOT NULL DEFAULT 0,
    theme       TEXT,
    PRIMARY KEY (user_id, interest_id)
);

-- Popular cities catalog (location picker).
CREATE TABLE cities (
    id           TEXT PRIMARY KEY,
    name         TEXT NOT NULL,
    country_code TEXT NOT NULL,
    lat          DOUBLE PRECISION NOT NULL,
    lng          DOUBLE PRECISION NOT NULL,
    theme        TEXT NOT NULL,
    position     INT  NOT NULL DEFAULT 0
);

INSERT INTO cities (id, name, country_code, lat, lng, theme, position) VALUES
 ('delhi',      'Delhi NCR',  'IN', 28.6139, 77.2090, 'DESERT', 0),
 ('mumbai',     'Mumbai',     'IN', 19.0760, 72.8777, 'OCEAN',  1),
 ('kolkata',    'Kolkata',    'IN', 22.5726, 88.3639, 'EMBER',  2),
 ('bengaluru',  'Bengaluru',  'IN', 12.9716, 77.5946, 'FOREST', 3),
 ('hyderabad',  'Hyderabad',  'IN', 17.3850, 78.4867, 'AURORA', 4),
 ('chandigarh', 'Chandigarh', 'IN', 30.7333, 76.7794, 'ALPINE', 5),
 ('pune',       'Pune',       'IN', 18.5204, 73.8567, 'COSMIC', 6),
 ('goa',        'Goa',        'IN', 15.2993, 74.1240, 'OCEAN',  7);

-- Popular search suggestions (search screen).
CREATE TABLE popular_searches (
    id       SERIAL PRIMARY KEY,
    query    TEXT NOT NULL,
    position INT  NOT NULL DEFAULT 0
);

INSERT INTO popular_searches (query, position) VALUES
 ('Trekking trails', 0),
 ('Sunrise viewpoints', 1),
 ('Climbing gyms', 2),
 ('Cafés to work from', 3),
 ('Live music', 4),
 ('Art & pottery workshops', 5),
 ('Weekend getaways', 6),
 ('Camping spots', 7);
