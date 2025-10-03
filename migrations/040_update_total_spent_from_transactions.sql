-- Update total_spent in users table based on successful transactions
-- This migration calculates total_spent by summing amount from transactions where status = 'success'

-- First, let's see what status values exist in transactions table
-- (This is just for reference, we'll use 'success' as requested)

-- Update total_spent for all users based on their successful transactions
UPDATE users 
SET total_spent = COALESCE(
    (
        SELECT SUM(amount) 
        FROM transactions 
        WHERE transactions.user_id = users.id 
        AND transactions.status = 'success'
    ), 
    0
);

-- Add comment to document this update
COMMENT ON COLUMN users.total_spent IS 'Total amount spent by user, calculated from successful transactions';

-- Optional: Create a view for easier querying of user spending
CREATE OR REPLACE VIEW user_spending_summary AS
SELECT 
    u.id as user_id,
    u.email,
    u.full_name,
    u.total_spent,
    COALESCE(SUM(t.amount), 0) as calculated_total_spent,
    COUNT(t.id) as successful_transactions_count
FROM users u
LEFT JOIN transactions t ON u.id = t.user_id AND t.status = 'success'
GROUP BY u.id, u.email, u.full_name, u.total_spent;

-- Add comment to the view
COMMENT ON VIEW user_spending_summary IS 'Summary of user spending based on successful transactions';
