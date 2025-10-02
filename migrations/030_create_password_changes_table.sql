-- Create password_changes table to track password change frequency
CREATE TABLE IF NOT EXISTS password_changes (
    id VARCHAR(50) PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create index for efficient querying by user_id and date
CREATE INDEX IF NOT EXISTS idx_password_changes_user_id_created_at ON password_changes (user_id, created_at);

-- Add comments
COMMENT ON TABLE password_changes IS 'Tracks password changes to enforce monthly limits';
COMMENT ON COLUMN password_changes.id IS 'Unique identifier for password change record';
COMMENT ON COLUMN password_changes.user_id IS 'User who changed their password';
COMMENT ON COLUMN password_changes.created_at IS 'When the password was changed';
