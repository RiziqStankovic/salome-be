-- Add soft delete columns to groups table
ALTER TABLE groups ADD COLUMN IF NOT EXISTS is_deleted BOOLEAN DEFAULT FALSE;
ALTER TABLE groups ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP;

-- Create index for better performance on soft delete queries
CREATE INDEX IF NOT EXISTS idx_groups_is_deleted ON groups(is_deleted);
CREATE INDEX IF NOT EXISTS idx_groups_deleted_at ON groups(deleted_at);
