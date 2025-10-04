-- Add necessary columns to transactions table for top-up functionality

-- Add transaction_type column
ALTER TABLE transactions 
ADD COLUMN IF NOT EXISTS transaction_type VARCHAR(20) DEFAULT 'group_payment';

-- Add payment_link_id column
ALTER TABLE transactions 
ADD COLUMN IF NOT EXISTS payment_link_id VARCHAR(255);

-- Add description column
ALTER TABLE transactions 
ADD COLUMN IF NOT EXISTS description TEXT;

-- Update existing transactions to have 'group_payment' type
UPDATE transactions 
SET transaction_type = 'group_payment' 
WHERE transaction_type IS NULL;

-- Add indexes for better query performance
CREATE INDEX IF NOT EXISTS idx_transactions_type ON transactions(transaction_type);
CREATE INDEX IF NOT EXISTS idx_transactions_user_type ON transactions(user_id, transaction_type);
CREATE INDEX IF NOT EXISTS idx_transactions_payment_link_id ON transactions(payment_link_id);

-- Add comments
COMMENT ON COLUMN transactions.transaction_type IS 'Type of transaction: top_up or group_payment';
COMMENT ON COLUMN transactions.payment_link_id IS 'Midtrans payment link ID for the transaction';
COMMENT ON COLUMN transactions.description IS 'Transaction description';