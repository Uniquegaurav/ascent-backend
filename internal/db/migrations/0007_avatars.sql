-- User avatar images, stored in the (persistent) database and served via /avatars/{id}.
CREATE TABLE avatars (
    user_id      UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    data         BYTEA NOT NULL,
    content_type TEXT NOT NULL DEFAULT 'image/jpeg',
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
