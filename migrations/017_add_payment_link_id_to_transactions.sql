-- Add payment_link_id column to transactions table
ALTER TABLE transactions 
ADD COLUMN IF NOT EXISTS payment_link_id VARCHAR(255);

-- Create index for better performance
CREATE INDEX IF NOT EXISTS idx_transactions_payment_link_id ON transactions(payment_link_id);
