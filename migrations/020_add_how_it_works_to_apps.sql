-- Add how_it_works column to apps table
ALTER TABLE apps ADD COLUMN IF NOT EXISTS how_it_works TEXT;

-- Add index for better performance
CREATE INDEX IF NOT EXISTS idx_apps_how_it_works ON apps(how_it_works);

-- Update existing apps with default how_it_works content
UPDATE apps SET how_it_works = '[
  "Create or join a group with friends",
  "One person becomes the group host", 
  "Host subscribes to the service",
  "Share login credentials securely",
  "Split the cost equally among members",
  "Enjoy premium features together!"
]' WHERE how_it_works IS NULL;
