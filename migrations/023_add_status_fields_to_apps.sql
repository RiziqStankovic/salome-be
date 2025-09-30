-- Add status fields to apps table
ALTER TABLE apps ADD COLUMN IF NOT EXISTS is_active BOOLEAN DEFAULT TRUE;
ALTER TABLE apps ADD COLUMN IF NOT EXISTS is_available BOOLEAN DEFAULT TRUE;

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_apps_is_active ON apps(is_active);
CREATE INDEX IF NOT EXISTS idx_apps_is_available ON apps(is_available);

-- Add constraints
ALTER TABLE apps ADD CONSTRAINT check_is_active CHECK (is_active IN (TRUE, FALSE));
ALTER TABLE apps ADD CONSTRAINT check_is_available CHECK (is_available IN (TRUE, FALSE));
