-- Update apps with proper pricing data
UPDATE apps SET 
    base_price = 150000,
    admin_fee_percentage = 10,
    is_active = true,
    is_available = true
WHERE id = 'netflix';

UPDATE apps SET 
    base_price = 200000,
    admin_fee_percentage = 10,
    is_active = true,
    is_available = true
WHERE id = 'disney_plus';

UPDATE apps SET 
    base_price = 120000,
    admin_fee_percentage = 10,
    is_active = true,
    is_available = true
WHERE id = 'spotify';

UPDATE apps SET 
    base_price = 180000,
    admin_fee_percentage = 10,
    is_active = true,
    is_available = true
WHERE id = 'youtube_premium';

UPDATE apps SET 
    base_price = 100000,
    admin_fee_percentage = 10,
    is_active = true,
    is_available = true
WHERE id = 'apple_music';

UPDATE apps SET 
    base_price = 250000,
    admin_fee_percentage = 10,
    is_active = true,
    is_available = true
WHERE id = 'canva';

UPDATE apps SET 
    base_price = 300000,
    admin_fee_percentage = 10,
    is_active = true,
    is_available = true
WHERE id = 'adobe_creative';

UPDATE apps SET 
    base_price = 80000,
    admin_fee_percentage = 10,
    is_active = true,
    is_available = true
WHERE id = 'office_365';

UPDATE apps SET 
    base_price = 160000,
    admin_fee_percentage = 10,
    is_active = true,
    is_available = true
WHERE id = 'calm';

UPDATE apps SET 
    base_price = 140000,
    admin_fee_percentage = 10,
    is_active = true,
    is_available = true
WHERE id = 'headspace';
