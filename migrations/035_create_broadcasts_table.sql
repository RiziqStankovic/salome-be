-- Create broadcasts table for admin announcements
CREATE TABLE IF NOT EXISTS broadcasts (
    id VARCHAR(50) PRIMARY KEY,
    admin_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    target_type VARCHAR(20) NOT NULL CHECK (target_type IN ('all_groups', 'selected_groups')),
    target_group_ids TEXT[], -- Array of group IDs for selected_groups type
    is_active BOOLEAN DEFAULT TRUE,
    priority INTEGER DEFAULT 1, -- 1 = normal, 2 = high, 3 = urgent
    start_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    end_date TIMESTAMP, -- NULL means no expiration
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_broadcasts_admin_id ON broadcasts(admin_id);
CREATE INDEX IF NOT EXISTS idx_broadcasts_target_type ON broadcasts(target_type);
CREATE INDEX IF NOT EXISTS idx_broadcasts_is_active ON broadcasts(is_active);
CREATE INDEX IF NOT EXISTS idx_broadcasts_start_date ON broadcasts(start_date);
CREATE INDEX IF NOT EXISTS idx_broadcasts_end_date ON broadcasts(end_date);
CREATE INDEX IF NOT EXISTS idx_broadcasts_priority ON broadcasts(priority);

-- Add comments
COMMENT ON TABLE broadcasts IS 'Admin broadcast messages to groups';
COMMENT ON COLUMN broadcasts.id IS 'Unique identifier for broadcast';
COMMENT ON COLUMN broadcasts.admin_id IS 'Admin user who created the broadcast';
COMMENT ON COLUMN broadcasts.title IS 'Broadcast title';
COMMENT ON COLUMN broadcasts.message IS 'Broadcast message content';
COMMENT ON COLUMN broadcasts.target_type IS 'Target type: all_groups or selected_groups';
COMMENT ON COLUMN broadcasts.target_group_ids IS 'Array of group IDs for selected_groups type';
COMMENT ON COLUMN broadcasts.is_active IS 'Whether the broadcast is currently active';
COMMENT ON COLUMN broadcasts.priority IS 'Broadcast priority: 1=normal, 2=high, 3=urgent';
COMMENT ON COLUMN broadcasts.start_date IS 'When the broadcast becomes active';
COMMENT ON COLUMN broadcasts.end_date IS 'When the broadcast expires (NULL = no expiration)';
COMMENT ON COLUMN broadcasts.created_at IS 'When the broadcast was created';
COMMENT ON COLUMN broadcasts.updated_at IS 'When the broadcast was last updated';
