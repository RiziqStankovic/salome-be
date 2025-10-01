-- Remove is_available column from apps table
ALTER TABLE apps DROP COLUMN IF EXISTS is_available;

-- Remove index if exists
DROP INDEX IF EXISTS idx_apps_is_available;

-- Remove constraint if exists
ALTER TABLE apps DROP CONSTRAINT IF EXISTS check_is_available;
