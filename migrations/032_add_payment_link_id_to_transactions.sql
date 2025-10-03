-- Add payment_link_id column to transactions table
ALTER TABLE transactions ADD COLUMN IF NOT EXISTS payment_link_id VARCHAR(255);

-- Add index for payment_link_id
CREATE INDEX IF NOT EXISTS idx_transactions_payment_link_id ON transactions (payment_link_id);

-- Add comment
COMMENT ON COLUMN transactions.payment_link_id IS 'Midtrans payment link ID for pending transactions';
