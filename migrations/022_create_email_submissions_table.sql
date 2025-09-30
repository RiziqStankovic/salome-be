-- Create email_submissions table
CREATE TABLE IF NOT EXISTS public.email_submissions (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    group_id varchar(255) NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    app_id varchar(255) NOT NULL REFERENCES apps(id) ON DELETE CASCADE,
    email varchar(255) NOT NULL,
    username varchar(255) NULL,
    full_name varchar(255) NOT NULL,
    status varchar(50) DEFAULT 'pending' NOT NULL CHECK (status IN ('pending', 'approved', 'rejected')),
    submitted_at timestamp DEFAULT CURRENT_TIMESTAMP NOT NULL,
    reviewed_at timestamp NULL,
    reviewed_by uuid NULL REFERENCES users(id) ON DELETE SET NULL,
    notes text NULL,
    created_at timestamp DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamp DEFAULT CURRENT_TIMESTAMP NOT NULL,
    CONSTRAINT email_submissions_pkey PRIMARY KEY (id),
    CONSTRAINT unique_user_group_submission UNIQUE (user_id, group_id)
);

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_email_submissions_user_id ON public.email_submissions(user_id);
CREATE INDEX IF NOT EXISTS idx_email_submissions_group_id ON public.email_submissions(group_id);
CREATE INDEX IF NOT EXISTS idx_email_submissions_app_id ON public.email_submissions(app_id);
CREATE INDEX IF NOT EXISTS idx_email_submissions_status ON public.email_submissions(status);
CREATE INDEX IF NOT EXISTS idx_email_submissions_submitted_at ON public.email_submissions(submitted_at);

-- Add comments
COMMENT ON TABLE public.email_submissions IS 'Email submissions for group applications';
COMMENT ON COLUMN public.email_submissions.id IS 'Unique identifier for email submission';
COMMENT ON COLUMN public.email_submissions.user_id IS 'User who submitted the email';
COMMENT ON COLUMN public.email_submissions.group_id IS 'Group for which email is submitted';
COMMENT ON COLUMN public.email_submissions.app_id IS 'Application for which email is submitted';
COMMENT ON COLUMN public.email_submissions.email IS 'Email address to be registered for the group';
COMMENT ON COLUMN public.email_submissions.username IS 'Username for the application (optional)';
COMMENT ON COLUMN public.email_submissions.full_name IS 'Full name to be registered for the group';
COMMENT ON COLUMN public.email_submissions.status IS 'Status of the submission (pending, approved, rejected)';
COMMENT ON COLUMN public.email_submissions.submitted_at IS 'When the submission was made';
COMMENT ON COLUMN public.email_submissions.reviewed_at IS 'When the submission was reviewed';
COMMENT ON COLUMN public.email_submissions.reviewed_by IS 'Admin who reviewed the submission';
COMMENT ON COLUMN public.email_submissions.notes IS 'Admin notes about the submission';
