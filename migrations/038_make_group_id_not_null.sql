

-- Make group_id NOT NULL in account_credentials table
-- First, update any NULL values to a default group ID if needed
-- For existing data, you might need to handle NULL group_id values

-- Update NULL group_id values to a default group (if any exist)
-- UPDATE account_credentials SET group_id = '00000000-0000-0000-0000-000000000000' WHERE group_id IS NULL;

-- Make group_id NOT NULL
ALTER TABLE account_credentials ALTER COLUMN group_id SET NOT NULL;

-- Add foreign key constraint if it doesn't exist
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_account_credentials_group') THEN
        ALTER TABLE account_credentials ADD CONSTRAINT fk_account_credentials_group
            FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE CASCADE;
    END IF;
END
$$;
