-- Create account_credentials table
CREATE TABLE IF NOT EXISTS public.account_credentials (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id uuid NOT NULL,
    app_id varchar(255) NOT NULL,
    username varchar(255) NULL,
    email varchar(255) NULL,
    created_at timestamp DEFAULT CURRENT_TIMESTAMP NULL,
    updated_at timestamp DEFAULT CURRENT_TIMESTAMP NULL,
    CONSTRAINT account_credentials_pkey PRIMARY KEY (id),
    CONSTRAINT fk_account_credentials_user_id FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_account_credentials_app_id FOREIGN KEY (app_id) REFERENCES apps(id) ON DELETE CASCADE,
    CONSTRAINT unique_user_app_credentials UNIQUE (user_id, app_id)
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_account_credentials_user_id ON account_credentials(user_id);
CREATE INDEX IF NOT EXISTS idx_account_credentials_app_id ON account_credentials(app_id);
CREATE INDEX IF NOT EXISTS idx_account_credentials_email ON account_credentials(email);

-- Add comments
COMMENT ON TABLE account_credentials IS 'Stores user account credentials for different apps';
COMMENT ON COLUMN account_credentials.user_id IS 'Reference to users table';
COMMENT ON COLUMN account_credentials.app_id IS 'Reference to apps table';
COMMENT ON COLUMN account_credentials.username IS 'Username for the app account';
COMMENT ON COLUMN account_credentials.email IS 'Email for the app account';

