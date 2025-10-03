-- Add is_read column to chats table for admin read tracking
ALTER TABLE chats ADD COLUMN IF NOT EXISTS is_read BOOLEAN DEFAULT false;

-- Add index for faster lookups
CREATE INDEX IF NOT EXISTS idx_chats_is_read ON chats(is_read);

-- Update existing chats to be marked as read
UPDATE chats SET is_read = true WHERE is_read IS NULL;
