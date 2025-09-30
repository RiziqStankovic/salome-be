-- Update specific apps with custom how_it_works content
-- This script shows examples of how to customize how_it_works for different apps

-- Netflix example
UPDATE apps SET how_it_works = '[
  "Create a Netflix group with up to 4 friends",
  "One person becomes the account owner",
  "Owner subscribes to Netflix Premium plan",
  "Share login credentials securely via SALOME",
  "Split the monthly cost equally (4 people)",
  "Enjoy unlimited streaming on all devices!"
]' WHERE name ILIKE '%netflix%';

-- Spotify example  
UPDATE apps SET how_it_works = '[
  "Create a Spotify group with up to 6 friends",
  "One person becomes the family plan manager",
  "Manager subscribes to Spotify Family plan",
  "Invite family members via email invitation",
  "Each member gets their own premium account",
  "Enjoy ad-free music and offline downloads!"
]' WHERE name ILIKE '%spotify%';

-- YouTube Premium example
UPDATE apps SET how_it_works = '[
  "Create a YouTube Premium group with up to 5 friends",
  "One person becomes the family plan manager", 
  "Manager subscribes to YouTube Premium Family",
  "Add family members through Google Family settings",
  "Each member gets their own YouTube Premium access",
  "Enjoy ad-free videos and background play!"
]' WHERE name ILIKE '%youtube%';

-- Canva example
UPDATE apps SET how_it_works = '[
  "Create a Canva group with up to 5 team members",
  "One person becomes the team owner",
  "Owner subscribes to Canva Pro for teams",
  "Invite team members via email invitation",
  "Share team access and brand assets",
  "Create professional designs together!"
]' WHERE name ILIKE '%canva%';

-- Adobe Creative Cloud example
UPDATE apps SET how_it_works = '[
  "Create an Adobe group with up to 2 friends",
  "One person becomes the account owner",
  "Owner subscribes to Adobe Creative Cloud",
  "Share login credentials securely",
  "Split the monthly subscription cost",
  "Access all Adobe creative tools and cloud storage!"
]' WHERE name ILIKE '%adobe%';
