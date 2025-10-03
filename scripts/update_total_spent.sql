-- Script to manually update total_spent for all users
-- This script calculates total_spent from successful transactions

-- First, let's see the current state
SELECT 
    u.id,
    u.email,
    u.full_name,
    u.total_spent as current_total_spent,
    COALESCE(SUM(t.amount), 0) as calculated_total_spent,
    COUNT(t.id) as successful_transactions_count
FROM users u
LEFT JOIN transactions t ON u.id = t.user_id AND t.status = 'success'
GROUP BY u.id, u.email, u.full_name, u.total_spent
ORDER BY u.created_at;

-- Update total_spent for all users
UPDATE users 
SET total_spent = COALESCE(
    (
        SELECT SUM(amount) 
        FROM transactions 
        WHERE transactions.user_id = users.id 
        AND transactions.status = 'success'
    ), 
    0
),
updated_at = NOW();

-- Verify the update
SELECT 
    u.id,
    u.email,
    u.full_name,
    u.total_spent,
    COALESCE(SUM(t.amount), 0) as calculated_total_spent,
    COUNT(t.id) as successful_transactions_count
FROM users u
LEFT JOIN transactions t ON u.id = t.user_id AND t.status = 'success'
GROUP BY u.id, u.email, u.full_name, u.total_spent
ORDER BY u.created_at;
