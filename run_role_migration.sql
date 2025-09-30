Update existing group members to have proper roles
-- Set owner role for group owners
UPDATE group_members 
SET role = 'owner' 
WHERE user_id IN (
    SELECT owner_id FROM groups WHERE groups.id = group_members.group_id
);

-- Set member role for all other members (where role is NULL)
UPDATE group_members 
SET role = 'member' 
WHERE role IS NULL;

-- Verify the update
SELECT 
    gm.id, 
    gm.user_id, 
    gm.role, 
    u.full_name,
    g.name as group_name,
    CASE 
        WHEN gm.user_id = g.owner_id THEN 'Should be owner'
        ELSE 'Should be member'
    END as expected_role
FROM group_members gm
JOIN users u ON gm.user_id = u.id
JOIN groups g ON gm.group_id = g.id
ORDER BY g.name, gm.role;
