-- Social graph + challenges + demo climbers to populate the feed/friends.

ALTER TABLE users ADD COLUMN phone_hash TEXT;
ALTER TABLE users ADD COLUMN is_demo BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE users ADD COLUMN last_ascent TEXT;

CREATE TABLE friendships (
    user_id  UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    other_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status   TEXT NOT NULL,
    PRIMARY KEY (user_id, other_id)
);

CREATE TABLE challenges (
    id            TEXT PRIMARY KEY,
    title         TEXT NOT NULL,
    description   TEXT NOT NULL,
    emoji         TEXT NOT NULL,
    interest_id   TEXT NOT NULL REFERENCES interests(id),
    target        INT NOT NULL,
    base_progress INT NOT NULL DEFAULT 0,
    unit          TEXT NOT NULL
);

CREATE TABLE user_challenges (
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    challenge_id TEXT NOT NULL REFERENCES challenges(id),
    joined       BOOLEAN NOT NULL DEFAULT TRUE,
    progress     INT NOT NULL DEFAULT 0,
    PRIMARY KEY (user_id, challenge_id)
);

-- Demo climbers (phone_hash mirrors the app's demo contacts so sync matches).
INSERT INTO users (id, phone, name, onboarded, avatar_hue, is_demo, phone_hash, last_ascent) VALUES
 ('11111111-1111-1111-1111-111111111111','+10000000001','Aria',  TRUE, 320, TRUE, 'h_aria',  'Bouldering V4 send'),
 ('22222222-2222-2222-2222-222222222222','+10000000002','Kabir', TRUE, 150, TRUE, NULL,      'Sunrise summit'),
 ('33333333-3333-3333-3333-333333333333','+10000000003','Mira',  TRUE, 45,  TRUE, 'h_mira',  'Coastal trail run'),
 ('44444444-4444-4444-4444-444444444444','+10000000004','Devon', TRUE, 205, TRUE, 'h_devon', 'Logged a sunrise hike');

INSERT INTO user_interests (user_id, interest_id) VALUES
 ('11111111-1111-1111-1111-111111111111','fitness'),
 ('11111111-1111-1111-1111-111111111111','trekking'),
 ('22222222-2222-2222-2222-222222222222','trekking'),
 ('33333333-3333-3333-3333-333333333333','running'),
 ('44444444-4444-4444-4444-444444444444','trekking');

INSERT INTO experiences (user_id, interest_id, title, description, category, mood_score, location_name, created_at) VALUES
 ('11111111-1111-1111-1111-111111111111','fitness','Bouldering V4 send','Crimps for days.','Fitness',5,'Hampi', now() - interval '4 hours'),
 ('22222222-2222-2222-2222-222222222222','trekking','Sunrise summit','Beat the sun to the top.','Trekking',5,'Triund', now() - interval '1 day'),
 ('33333333-3333-3333-3333-333333333333','running','Coastal trail run','Salt air, soft sand.','Running',4,'Lisbon', now() - interval '2 days'),
 ('44444444-4444-4444-4444-444444444444','trekking','Sunrise hike','Cold start, warm view.','Trekking',4,'Nandi Hills', now() - interval '3 days');

-- Challenge catalog.
INSERT INTO challenges (id, title, description, emoji, interest_id, target, base_progress, unit) VALUES
 ('c_trek','Summit 3 new peaks','Top out three you haven''t yet.','🏔️','trekking',3,1,'peaks'),
 ('c_cities','Wander 3 new places','Expand your map.','🗺️','travel',3,1,'places'),
 ('c_books','Read 2 books','Feed the cosmic peak.','📚','reading',2,0,'books'),
 ('c_run','Run a coastal 50K','One trail at a time.','🏃','running',50,31,'km');
