-- Prevent adding the same explore item to a user's ascents twice.
CREATE UNIQUE INDEX IF NOT EXISTS ascents_user_source_uniq
    ON ascents (user_id, source_item_id)
    WHERE source_item_id IS NOT NULL;
