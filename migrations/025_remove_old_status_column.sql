-- Remove old status column from groups table since we now use group_status
-- This migration removes the old 'status' column that was added in migration 006

-- First, check if the old status column exists and drop it
DO $$ 
BEGIN
    IF EXISTS (
        SELECT 1 
        FROM information_schema.columns 
        WHERE table_name = 'groups' 
        AND column_name = 'status'
    ) THEN
        ALTER TABLE groups DROP COLUMN status;
        RAISE NOTICE 'Dropped old status column from groups table';
    ELSE
        RAISE NOTICE 'Old status column does not exist in groups table';
    END IF;
END $$;
