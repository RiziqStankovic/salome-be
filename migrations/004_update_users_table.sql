-- Update users table to add balance and total_spent columns
ALTER TABLE users ADD COLUMN IF NOT EXISTS balance DECIMAL(15,2) DEFAULT 0.00;
ALTER TABLE users ADD COLUMN IF NOT EXISTS total_spent DECIMAL(15,2) DEFAULT 0.00;
