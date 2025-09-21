-- Update group_members table to add new columns according to ERD
ALTER TABLE group_members ADD COLUMN IF NOT EXISTS status VARCHAR(20) DEFAULT 'pending';
ALTER TABLE group_members ADD COLUMN IF NOT EXISTS payment_amount DECIMAL(15,2) DEFAULT 0.00;
