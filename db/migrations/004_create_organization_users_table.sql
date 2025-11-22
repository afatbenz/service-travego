-- Create organizations table
CREATE TABLE IF NOT EXISTS organization_users (
    uuid uuid NOT NULL,
    user_id uuid NOT NULL,
    organization_id uuid NOT NULL,
    role_user integer NOT NULL,
    is_active boolean,
    created_at timestamp with time zone,
    created_by uuid NOT NULL,
    updated_at timestamp with time zone,
    updated_by uuid NOT NULL,
    PRIMARY KEY (uuid),
    FOREIGN KEY (user_id) REFERENCES public.users(user_id) ON DELETE CASCADE,
    FOREIGN KEY (organization_id) REFERENCES public.organizations(organization_id) ON DELETE CASCADE
);

-- Create index on organization_code for faster lookups
CREATE INDEX IF NOT EXISTS idx_organization_users_user_id ON organization_users(user_id);
CREATE INDEX IF NOT EXISTS idx_organization_users_organization_id ON organization_users(organization_id);
CREATE INDEX IF NOT EXISTS idx_organization_users_created_by ON organization_users(created_by);
CREATE INDEX IF NOT EXISTS idx_organization_users_updated_by ON organization_users(updated_by);