-- Rename columns in apps table
-- total_members -> max_group_members
-- base_price -> total_price

-- First, drop the old columns if they exist and add new ones
ALTER TABLE apps DROP COLUMN IF EXISTS total_members;
ALTER TABLE apps DROP COLUMN IF EXISTS base_price;

-- Add the new columns with the correct names
ALTER TABLE apps ADD COLUMN IF NOT EXISTS max_group_members INTEGER DEFAULT 5;
ALTER TABLE apps ADD COLUMN IF NOT EXISTS total_price DECIMAL(15,2) DEFAULT 0.00;

-- Update existing data if needed (copy from old columns if they still exist)
-- This is a safety measure in case the columns still exist
UPDATE apps SET 
    max_group_members = COALESCE(max_group_members, 5),
    total_price = COALESCE(total_price, 0.00)
WHERE max_group_members IS NULL OR total_price IS NULL;
