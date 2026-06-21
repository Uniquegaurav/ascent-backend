-- Activity-aware log fields (e.g. running distance/pace, gym exercises) as a flexible
-- label→value map. Summit/Explore logs simply leave it empty.
ALTER TABLE logs ADD COLUMN metrics JSONB NOT NULL DEFAULT '{}'::jsonb;
