-- Check if apps table exists and has data
SELECT COUNT(*) as total_apps FROM apps;

-- Check if apps table has required columns
SELECT column_name, data_type, is_nullable 
FROM information_schema.columns 
WHERE table_name = 'apps' 
ORDER BY ordinal_position;

-- Check if there are any active apps
SELECT COUNT(*) as active_apps 
FROM apps 
WHERE is_active = true AND is_available = true;

-- Sample data check
SELECT id, name, is_active, is_available, how_it_works
FROM apps 
LIMIT 3;
