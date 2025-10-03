-- salome-be/migrations/043_create_user_broadcast_table.sql
CREATE TABLE IF NOT EXISTS user_broadcast (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    target_type VARCHAR(50) NOT NULL DEFAULT 'all', -- 'all' or 'selected'
    priority VARCHAR(20) NOT NULL DEFAULT 'normal', -- 'low', 'normal', 'high'
    status VARCHAR(20) NOT NULL DEFAULT 'draft', -- 'draft', 'scheduled', 'sent', 'cancelled'
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    scheduled_at TIMESTAMP WITH TIME ZONE,
    sent_at TIMESTAMP WITH TIME ZONE,
    end_date TIMESTAMP WITH TIME ZONE,
    success_count INTEGER DEFAULT 0,
    error_count INTEGER DEFAULT 0,
    total_targets INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS user_broadcast_targets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    broadcast_id UUID NOT NULL REFERENCES user_broadcast(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL DEFAULT 'pending', -- 'pending', 'sent', 'failed'
    sent_at TIMESTAMP WITH TIME ZONE,
    error_message TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for better performance
CREATE INDEX IF NOT EXISTS idx_user_broadcast_created_by ON user_broadcast(created_by);
CREATE INDEX IF NOT EXISTS idx_user_broadcast_status ON user_broadcast(status);
CREATE INDEX IF NOT EXISTS idx_user_broadcast_scheduled_at ON user_broadcast(scheduled_at);
CREATE INDEX IF NOT EXISTS idx_user_broadcast_targets_broadcast_id ON user_broadcast_targets(broadcast_id);
CREATE INDEX IF NOT EXISTS idx_user_broadcast_targets_user_id ON user_broadcast_targets(user_id);
CREATE INDEX IF NOT EXISTS idx_user_broadcast_targets_status ON user_broadcast_targets(status);
