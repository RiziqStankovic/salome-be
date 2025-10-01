-- Update apps with proper pricing data
UPDATE apps SET 
    total_price = 150000,
    max_group_members = 4,
    admin_fee_percentage = 10
WHERE id = 'netflix';

UPDATE apps SET 
    total_price = 200000,
    max_group_members = 4,
    admin_fee_percentage = 10
WHERE id = 'disney_plus';

UPDATE apps SET 
    total_price = 120000,
    max_group_members = 6,
    admin_fee_percentage = 10
WHERE id = 'spotify';

UPDATE apps SET 
    total_price = 180000,
    max_group_members = 6,
    admin_fee_percentage = 10
WHERE id = 'youtube_premium';

UPDATE apps SET 
    total_price = 100000,
    max_group_members = 6,
    admin_fee_percentage = 10
WHERE id = 'apple_music';

UPDATE apps SET 
    total_price = 250000,
    max_group_members = 5,
    admin_fee_percentage = 10
WHERE id = 'canva';

UPDATE apps SET 
    total_price = 300000,
    max_group_members = 2,
    admin_fee_percentage = 10
WHERE id = 'adobe_creative';

UPDATE apps SET 
    total_price = 80000,
    max_group_members = 6,
    admin_fee_percentage = 10
WHERE id = 'office_365';

UPDATE apps SET 
    total_price = 160000,
    max_group_members = 4,
    admin_fee_percentage = 10
WHERE id = 'calm';

UPDATE apps SET 
    total_price = 140000,
    max_group_members = 4,
    admin_fee_percentage = 10
WHERE id = 'headspace';
