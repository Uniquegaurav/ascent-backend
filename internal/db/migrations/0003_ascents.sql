-- Explore-driven model: ascents (chosen pursuits) + logs (entries under them).

DROP TABLE IF EXISTS experiences CASCADE;

CREATE TABLE ascents (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title         TEXT NOT NULL,
    category      TEXT NOT NULL DEFAULT '',
    theme         TEXT NOT NULL DEFAULT 'ALPINE',
    image_url     TEXT NOT NULL DEFAULT '',
    kind          TEXT NOT NULL DEFAULT 'HOBBY',
    location_name TEXT,
    source_item_id TEXT,
    status        TEXT NOT NULL DEFAULT 'ACTIVE',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX ascents_user_idx ON ascents (user_id, created_at DESC);

CREATE TABLE logs (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ascent_id     UUID NOT NULL REFERENCES ascents(id) ON DELETE CASCADE,
    user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title         TEXT NOT NULL,
    note          TEXT NOT NULL DEFAULT '',
    mood_score    INT NOT NULL DEFAULT 4,
    location_name TEXT,
    lat           DOUBLE PRECISION,
    lng           DOUBLE PRECISION,
    image_urls    TEXT[] NOT NULL DEFAULT '{}',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX logs_user_idx ON logs (user_id, created_at DESC);
CREATE INDEX logs_ascent_idx ON logs (ascent_id, created_at DESC);

CREATE TABLE ascent_invites (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    from_user   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    to_user     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    ascent_id   UUID NOT NULL REFERENCES ascents(id) ON DELETE CASCADE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Seed demo climbers' ascents + a log each, so feed / friend-logs populate.
INSERT INTO ascents (id, user_id, title, category, theme, kind, location_name, status, image_url) VALUES
 ('a1111111-1111-1111-1111-111111111111','11111111-1111-1111-1111-111111111111','Bouldering at Hampi','fitness','ALPINE','PLACE','Hampi','ACTIVE','https://images.unsplash.com/photo-1522163182402-834f871fd851?auto=format&fit=crop&w=1200&q=80'),
 ('a2222222-2222-2222-2222-222222222222','22222222-2222-2222-2222-222222222222','Triund Trek','trekking','ALPINE','TREK','Triund','ACTIVE','https://images.unsplash.com/photo-1454496522488-7a8e488e8606?auto=format&fit=crop&w=1200&q=80'),
 ('a3333333-3333-3333-3333-333333333333','33333333-3333-3333-3333-333333333333','Coastal Running','running','OCEAN','HOBBY','Lisbon','ACTIVE','https://images.unsplash.com/photo-1502904550040-7534597429ae?auto=format&fit=crop&w=1200&q=80'),
 ('a4444444-4444-4444-4444-444444444444','44444444-4444-4444-4444-444444444444','Sunrise Hikes','trekking','FOREST','TREK','Nandi Hills','ACTIVE','https://images.unsplash.com/photo-1486870591958-9b9d0d1dda99?auto=format&fit=crop&w=1200&q=80');

INSERT INTO logs (ascent_id, user_id, title, note, mood_score, location_name, created_at) VALUES
 ('a1111111-1111-1111-1111-111111111111','11111111-1111-1111-1111-111111111111','Bouldering V4 send','Crimps for days.',5,'Hampi', now() - interval '4 hours'),
 ('a2222222-2222-2222-2222-222222222222','22222222-2222-2222-2222-222222222222','Sunrise summit','Beat the sun to the top.',5,'Triund', now() - interval '1 day'),
 ('a3333333-3333-3333-3333-333333333333','33333333-3333-3333-3333-333333333333','Coastal trail run','Salt air, soft sand.',4,'Lisbon', now() - interval '2 days'),
 ('a4444444-4444-4444-4444-444444444444','44444444-4444-4444-4444-444444444444','Sunrise hike','Cold start, warm view.',4,'Nandi Hills', now() - interval '3 days');
