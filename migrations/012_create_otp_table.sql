-- Create OTP table for 6-digit verification codes
CREATE TABLE IF NOT EXISTS public.otps (
    id varchar(50) NOT NULL,
    user_id uuid NOT NULL,
    email varchar(255) NOT NULL,
    otp_code varchar(6) NOT NULL,
    purpose varchar(50) NOT NULL, -- 'email_verification', 'password_reset', 'login_verification'
    expires_at timestamp NOT NULL,
    is_used boolean DEFAULT false,
    attempts integer DEFAULT 0,
    created_at timestamp DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT otps_pkey PRIMARY KEY (id),
    CONSTRAINT otps_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE,
    CONSTRAINT otps_purpose_check CHECK (((purpose)::text = ANY ((ARRAY['email_verification'::character varying, 'password_reset'::character varying, 'login_verification'::character varying])::text[]))),
    CONSTRAINT otps_otp_code_check CHECK ((char_length(otp_code) = 6))
);

-- Create index for faster lookups
CREATE INDEX IF NOT EXISTS idx_otps_user_id ON public.otps(user_id);
CREATE INDEX IF NOT EXISTS idx_otps_email ON public.otps(email);
CREATE INDEX IF NOT EXISTS idx_otps_code ON public.otps(otp_code);
CREATE INDEX IF NOT EXISTS idx_otps_expires_at ON public.otps(expires_at);
