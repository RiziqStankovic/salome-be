-- Add payment_reference column to transactions table
-- This column will store Midtrans order ID or payment reference

ALTER TABLE transactions 
ADD COLUMN IF NOT EXISTS payment_reference VARCHAR(255);

-- Add index for better query performance
CREATE INDEX IF NOT EXISTS idx_transactions_payment_reference ON transactions(payment_reference);

-- Add comment
COMMENT ON COLUMN transactions.payment_reference IS 'Midtrans order ID or payment reference for the transaction';
