-- Fix OTP rate limiting by ensuring columns exist
ALTER TABLE otps ADD COLUMN IF NOT EXISTS rate_limit_count INTEGER DEFAULT 0;
ALTER TABLE otps ADD COLUMN IF NOT EXISTS rate_limit_reset_at TIMESTAMP NULL;

-- Create indexes for rate limiting queries
CREATE INDEX IF NOT EXISTS idx_otps_rate_limit ON otps (email, rate_limit_reset_at);
CREATE INDEX IF NOT EXISTS idx_otps_user_rate_limit ON otps (user_id, rate_limit_reset_at);

-- Add comments
COMMENT ON COLUMN otps.rate_limit_count IS 'Number of OTP requests in current rate limit window';
COMMENT ON COLUMN otps.rate_limit_reset_at IS 'When the rate limit resets';
