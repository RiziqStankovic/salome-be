-- Update existing group members to have proper roles
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
