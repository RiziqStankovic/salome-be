-- Remove app_id column and make all fields required
-- Update account_credentials table structure

-- First, drop the existing foreign key constraint for app_id
ALTER TABLE account_credentials DROP CONSTRAINT IF EXISTS fk_account_credentials_app_id;

-- Remove app_id column
ALTER TABLE account_credentials DROP COLUMN IF EXISTS app_id;

-- Make group_id NOT NULL (required)
ALTER TABLE account_credentials ALTER COLUMN group_id SET NOT NULL;

-- Make username NOT NULL (required)
ALTER TABLE account_credentials ALTER COLUMN username SET NOT NULL;

-- Make email NOT NULL (required)
ALTER TABLE account_credentials ALTER COLUMN email SET NOT NULL;

-- Make description NOT NULL (required)
ALTER TABLE account_credentials ALTER COLUMN description SET NOT NULL;

-- Update comments
COMMENT ON TABLE account_credentials IS 'Account credentials for group members - all fields required';
COMMENT ON COLUMN account_credentials.group_id IS 'Group this credential belongs to (required)';
COMMENT ON COLUMN account_credentials.username IS 'Username for the account (required)';
COMMENT ON COLUMN account_credentials.email IS 'Email for the account (required)';
COMMENT ON COLUMN account_credentials.description IS 'Description of the account credential (required)';
