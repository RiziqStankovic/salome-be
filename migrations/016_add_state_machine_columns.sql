-- -- Add state machine columns for user and group lifecycle management

-- -- Add user status columns to group_members table
-- ALTER TABLE group_members ADD COLUMN IF NOT EXISTS user_status VARCHAR(20) DEFAULT 'pending';
-- ALTER TABLE group_members ADD COLUMN IF NOT EXISTS payment_deadline TIMESTAMP;
-- ALTER TABLE group_members ADD COLUMN IF NOT EXISTS paid_at TIMESTAMP;
-- ALTER TABLE group_members ADD COLUMN IF NOT EXISTS activated_at TIMESTAMP;
-- ALTER TABLE group_members ADD COLUMN IF NOT EXISTS expired_at TIMESTAMP;
-- ALTER TABLE group_members ADD COLUMN IF NOT EXISTS removed_at TIMESTAMP;
-- ALTER TABLE group_members ADD COLUMN IF NOT EXISTS removed_reason VARCHAR(100);

-- -- Add group status columns to groups table
-- ALTER TABLE groups ADD COLUMN IF NOT EXISTS group_status VARCHAR(20) DEFAULT 'open';
-- ALTER TABLE groups ADD COLUMN IF NOT EXISTS all_paid_at TIMESTAMP;
-- ALTER TABLE group_members ADD COLUMN IF NOT EXISTS subscription_period_start TIMESTAMP;
-- ALTER TABLE group_members ADD COLUMN IF NOT EXISTS subscription_period_end TIMESTAMP;

-- -- Add admin privileges column to users table
-- ALTER TABLE users ADD COLUMN IF NOT EXISTS is_admin BOOLEAN DEFAULT FALSE;

-- -- Create indexes for better performance
-- CREATE INDEX IF NOT EXISTS idx_group_members_user_status ON group_members(user_status);
-- CREATE INDEX IF NOT EXISTS idx_group_members_payment_deadline ON group_members(payment_deadline);
-- CREATE INDEX IF NOT EXISTS idx_groups_group_status ON groups(group_status);
-- CREATE INDEX IF NOT EXISTS idx_users_is_admin ON users(is_admin);

-- -- Add constraints
-- ALTER TABLE group_members ADD CONSTRAINT check_user_status 
--   CHECK (user_status IN ('pending', 'paid', 'active', 'expired', 'removed'));

-- ALTER TABLE groups ADD CONSTRAINT check_group_status 
--   CHECK (group_status IN ('open', 'private', 'full', 'paid_group', 'closed'));
