-- Create group_payments table
CREATE TABLE IF NOT EXISTS group_payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id UUID NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    total_collected DECIMAL(15,2) DEFAULT 0.00,
    total_required DECIMAL(15,2) DEFAULT 0.00,
    payment_status VARCHAR(20) DEFAULT 'pending',
    provider_purchase_id VARCHAR(100),
    provider_credentials VARCHAR(100),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_group_payments_group_id ON group_payments(group_id);
CREATE INDEX IF NOT EXISTS idx_group_payments_payment_status ON group_payments(payment_status);
