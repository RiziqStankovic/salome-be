-- Add role column to group_members table
ALTER TABLE group_members ADD COLUMN IF NOT EXISTS role VARCHAR(20) DEFAULT 'member';

-- Add constraint for role values
DO $$ 
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'check_role') THEN
        ALTER TABLE group_members ADD CONSTRAINT check_role CHECK (role IN ('owner', 'admin', 'member'));
    END IF;
END $$;

-- Update existing records
UPDATE group_members 
SET role = 'owner' 
WHERE user_id IN (
    SELECT owner_id FROM groups WHERE groups.id = group_members.group_id
);

UPDATE group_members 
SET role = 'member' 
WHERE role IS NULL;

-- Verify the results
SELECT 
    gm.id, 
    gm.user_id, 
    gm.role, 
    u.full_name,
    g.name as group_name
FROM group_members gm
JOIN users u ON gm.user_id = u.id
JOIN groups g ON gm.group_id = g.id
ORDER BY g.name, gm.role;
