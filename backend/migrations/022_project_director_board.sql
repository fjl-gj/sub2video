-- A director board is part of an existing creator-owned work, not a second
-- project/task system. Array order is the shot order; writes use a revision.
ALTER TABLE projects
  ADD COLUMN IF NOT EXISTS director_board JSONB NOT NULL DEFAULT '[]'::jsonb
    CHECK (jsonb_typeof(director_board) = 'array' AND jsonb_array_length(director_board) <= 100),
  ADD COLUMN IF NOT EXISTS director_revision BIGINT NOT NULL DEFAULT 0
    CHECK (director_revision >= 0);
