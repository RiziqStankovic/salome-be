-- Add balance and total_spent columns to users table

-- Add balance column (default 0)
ALTER TABLE users 
ADD COLUMN IF NOT EXISTS balance INTEGER DEFAULT 0;

-- Add total_spent column (default 0)
ALTER TABLE users 
ADD COLUMN IF NOT EXISTS total_spent INTEGER DEFAULT 0;

-- Update existing users to have 0 balance and total_spent
UPDATE users 
SET balance = 0, total_spent = 0 
WHERE balance IS NULL OR total_spent IS NULL;

-- Add comments
COMMENT ON COLUMN users.balance IS 'Current user balance in IDR';
COMMENT ON COLUMN users.total_spent IS 'Total amount spent by user in IDR';
 