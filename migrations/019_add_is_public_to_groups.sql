-- Add is_public column to groups table
ALTER TABLE groups ADD COLUMN IF NOT EXISTS is_public BOOLEAN DEFAULT TRUE;

-- Create index for better performance on public groups filtering
CREATE INDEX IF NOT EXISTS idx_groups_is_public ON groups(is_public);

-- Add constraint to ensure is_public is boolean
ALTER TABLE groups ADD CONSTRAINT check_is_public CHECK (is_public IN (TRUE, FALSE));
