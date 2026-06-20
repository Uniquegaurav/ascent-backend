CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE users (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    phone       TEXT UNIQUE NOT NULL,
    name        TEXT NOT NULL DEFAULT '',
    onboarded   BOOLEAN NOT NULL DEFAULT FALSE,
    avatar_hue  REAL NOT NULL DEFAULT 28,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE otp_codes (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    phone       TEXT NOT NULL,
    code_hash   TEXT NOT NULL,
    expires_at  TIMESTAMPTZ NOT NULL,
    consumed    BOOLEAN NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX otp_codes_phone_idx ON otp_codes (phone);

CREATE TABLE interests (
    id          TEXT PRIMARY KEY,
    label       TEXT NOT NULL,
    emoji       TEXT NOT NULL,
    theme       TEXT NOT NULL,
    image_url   TEXT NOT NULL DEFAULT '',
    vibe        TEXT NOT NULL
);

CREATE TABLE user_interests (
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    interest_id TEXT NOT NULL REFERENCES interests(id),
    PRIMARY KEY (user_id, interest_id)
);

CREATE TABLE experiences (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    interest_id   TEXT NOT NULL REFERENCES interests(id),
    title         TEXT NOT NULL,
    description   TEXT NOT NULL DEFAULT '',
    category      TEXT NOT NULL DEFAULT '',
    mood_score    INT NOT NULL DEFAULT 4,
    location_name TEXT,
    lat           DOUBLE PRECISION,
    lng           DOUBLE PRECISION,
    image_urls    TEXT[] NOT NULL DEFAULT '{}',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX experiences_user_idx ON experiences (user_id, created_at DESC);

-- Seed the interest catalog (mirrors the app's InterestCatalog).
INSERT INTO interests (id, label, emoji, theme, vibe, image_url) VALUES
 ('trekking','Trekking','🏔️','ALPINE','ADVENTURE','https://images.unsplash.com/photo-1454496522488-7a8e488e8606?auto=format&fit=crop&w=1200&q=80'),
 ('travel','Travel','✈️','DESERT','ADVENTURE','https://images.unsplash.com/photo-1488646953014-85cb44e25828?auto=format&fit=crop&w=1200&q=80'),
 ('adventure','Adventure Sports','🪂','FOREST','ADVENTURE','https://images.unsplash.com/photo-1605540436563-5bca919ae766?auto=format&fit=crop&w=1200&q=80'),
 ('photography','Photography','📸','OCEAN','ADVENTURE','https://images.unsplash.com/photo-1452780212940-6f5c0d14d848?auto=format&fit=crop&w=1200&q=80'),
 ('running','Running','🏃','FOREST','MOVEMENT','https://images.unsplash.com/photo-1502904550040-7534597429ae?auto=format&fit=crop&w=1200&q=80'),
 ('football','Football','⚽','EMBER','MOVEMENT','https://images.unsplash.com/photo-1431324155629-1a6deb1dec8d?auto=format&fit=crop&w=1200&q=80'),
 ('dance','Dance','💃','EMBER','MOVEMENT','https://images.unsplash.com/photo-1504609773096-104ff2c73ba4?auto=format&fit=crop&w=1200&q=80'),
 ('fitness','Fitness','🏋️','ALPINE','MOVEMENT','https://images.unsplash.com/photo-1534438327276-14e5300c3a48?auto=format&fit=crop&w=1200&q=80'),
 ('music','Music','🎸','EMBER','CREATIVE','https://images.unsplash.com/photo-1510915361894-db8b60106cb1?auto=format&fit=crop&w=1200&q=80'),
 ('art','Creating Art','🎨','AURORA','CREATIVE','https://images.unsplash.com/photo-1513364776144-60967b0f800f?auto=format&fit=crop&w=1200&q=80'),
 ('writing','Writing','✍️','COSMIC','CREATIVE','https://images.unsplash.com/photo-1455390582262-044cdead5f82?auto=format&fit=crop&w=1200&q=80'),
 ('reading','Reading','📚','COSMIC','MIND','https://images.unsplash.com/photo-1512820790803-83ca734da794?auto=format&fit=crop&w=1200&q=80'),
 ('learning','Learning Skills','🧠','COSMIC','MIND','https://images.unsplash.com/photo-1503676260728-1c00da094a0b?auto=format&fit=crop&w=1200&q=80'),
 ('gaming','Gaming','🎮','AURORA','MIND','https://images.unsplash.com/photo-1587202372775-e229f172b9d7?auto=format&fit=crop&w=1200&q=80'),
 ('community','Building Communities','🌐','AURORA','SOCIAL','https://images.unsplash.com/photo-1529156069898-49953e39b3ac?auto=format&fit=crop&w=1200&q=80');
