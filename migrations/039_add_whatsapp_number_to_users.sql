-- Add whatsapp_number column to users table
ALTER TABLE users ADD COLUMN IF NOT EXISTS whatsapp_number VARCHAR(20);

-- Add index for better performance on whatsapp_number queries
CREATE INDEX IF NOT EXISTS idx_users_whatsapp_number ON users(whatsapp_number);

-- Add comment to the column
COMMENT ON COLUMN users.whatsapp_number IS 'WhatsApp number for user contact';
