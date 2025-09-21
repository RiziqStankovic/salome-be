-- Add rate limiting columns to otps table
ALTER TABLE public.otps
ADD COLUMN IF NOT EXISTS rate_limit_count INTEGER DEFAULT 0,
ADD COLUMN IF NOT EXISTS rate_limit_reset_at TIMESTAMP NULL;

-- Create index for rate limiting queries
CREATE INDEX IF NOT EXISTS idx_otps_rate_limit ON public.otps (email, rate_limit_reset_at);

-- Create index for user rate limiting
CREATE INDEX IF NOT EXISTS idx_otps_user_rate_limit ON public.otps (user_id, rate_limit_reset_at);
