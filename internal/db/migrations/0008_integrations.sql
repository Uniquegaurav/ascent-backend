-- Connected third-party apps per user (Strava, Cult.fit, AllTrails, …).
CREATE TABLE user_integrations (
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider     TEXT NOT NULL,
    connected_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, provider)
);
