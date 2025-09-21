-- Add status column to users table for verification tracking
ALTER TABLE public.users 
ADD COLUMN IF NOT EXISTS status varchar(20) DEFAULT 'pending_verification' NOT NULL;

-- Add constraint for status values
ALTER TABLE public.users 
ADD CONSTRAINT users_status_check CHECK (
    status IN ('pending_verification', 'active', 'suspended', 'deleted')
);

-- Add index for faster status queries
CREATE INDEX IF NOT EXISTS idx_users_status ON public.users(status);

-- Update existing users to active status (if any)
UPDATE public.users 
SET status = 'active' 
WHERE status = 'pending_verification' AND is_verified = true;
