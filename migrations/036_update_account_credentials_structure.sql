-- Update account_credentials table structure
-- Change app_id to group_id and add description column

-- First, drop the existing foreign key constraint if it exists
ALTER TABLE account_credentials DROP CONSTRAINT IF EXISTS fk_account_credentials_app_id;

-- Add group_id column
ALTER TABLE account_credentials ADD COLUMN IF NOT EXISTS group_id UUID;

-- Add description column
ALTER TABLE account_credentials ADD COLUMN IF NOT EXISTS description TEXT;

-- Update existing records to have group_id (you may need to adjust this based on your data)
-- For now, we'll set group_id to NULL for existing records
-- You can update them manually based on your business logic

-- Add foreign key constraint for group_id
ALTER TABLE account_credentials 
ADD CONSTRAINT fk_account_credentials_group_id 
FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE CASCADE;

-- Add foreign key constraint for user_id if it doesn't exist
ALTER TABLE account_credentials 
ADD CONSTRAINT fk_account_credentials_user_id 
FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

-- Create index for better performance
CREATE INDEX IF NOT EXISTS idx_account_credentials_group_id ON account_credentials(group_id);
CREATE INDEX IF NOT EXISTS idx_account_credentials_user_id ON account_credentials(user_id);
CREATE INDEX IF NOT EXISTS idx_account_credentials_app_id ON account_credentials(app_id);

-- Add comments
COMMENT ON TABLE account_credentials IS 'Account credentials for group members';
COMMENT ON COLUMN account_credentials.id IS 'Unique identifier for account credential';
COMMENT ON COLUMN account_credentials.user_id IS 'User who owns this credential';
COMMENT ON COLUMN account_credentials.group_id IS 'Group this credential belongs to';
COMMENT ON COLUMN account_credentials.app_id IS 'Application this credential is for';
COMMENT ON COLUMN account_credentials.username IS 'Username for the account';
COMMENT ON COLUMN account_credentials.email IS 'Email for the account';
COMMENT ON COLUMN account_credentials.description IS 'Description of the account credential';
COMMENT ON COLUMN account_credentials.created_at IS 'When the credential was created';
COMMENT ON COLUMN account_credentials.updated_at IS 'When the credential was last updated';
