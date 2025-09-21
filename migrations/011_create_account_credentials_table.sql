-- Create account_credentials table
CREATE TABLE IF NOT EXISTS account_credentials (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    subscription_id UUID NOT NULL REFERENCES subscriptions(id) ON DELETE CASCADE,
    username VARCHAR(255),
    password_encrypted TEXT,
    additional_info JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_account_credentials_subscription_id ON account_credentials(subscription_id);
