-- Prevent adding the same explore item to a user's ascents twice.

-- First collapse any pre-existing duplicates (kept the earliest row); their logs
-- cascade-delete. This makes the unique index below safe on existing data.
DELETE FROM ascents a
USING ascents b
WHERE a.user_id = b.user_id
  AND a.source_item_id = b.source_item_id
  AND a.source_item_id IS NOT NULL
  AND a.ctid > b.ctid;

CREATE UNIQUE INDEX IF NOT EXISTS ascents_user_source_uniq
    ON ascents (user_id, source_item_id)
    WHERE source_item_id IS NOT NULL;
