-- Create apps table
CREATE TABLE IF NOT EXISTS apps (
    id VARCHAR(50) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    category VARCHAR(50),
    icon_url TEXT,
    website_url TEXT,
    total_members INTEGER DEFAULT 0,
    total_price INTEGER DEFAULT 0,
    is_popular BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create index for better performance
CREATE INDEX IF NOT EXISTS idx_apps_category ON apps(category);
CREATE INDEX IF NOT EXISTS idx_apps_popular ON apps(is_popular);
CREATE INDEX IF NOT EXISTS idx_apps_name ON apps(name);
