-- Create organizations table
CREATE TABLE IF NOT EXISTS organizations (
    organization_id uuid NOT NULL,
    organization_code VARCHAR(10) UNIQUE NOT NULL,
    organization_name VARCHAR(255) NOT NULL,
    company_name VARCHAR(255) NOT NULL,
    address character varying(100),
    city character varying(100),
    province character varying(30),
    phone character varying(20),
    npwp_number character varying(30),
    email character varying(50),
    created_by uuid NOT NULL,
    organization_type integer NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    PRIMARY KEY (organization_id),
    FOREIGN KEY (created_by) REFERENCES public.users(user_id) ON DELETE CASCADE
);

-- Create index on organization_code for faster lookups
CREATE INDEX IF NOT EXISTS idx_organizations_code ON organizations(organization_code);
CREATE INDEX IF NOT EXISTS idx_organizations_created_by ON organizations(created_by);

-- For MySQL (use this if using MySQL)
-- CREATE TABLE IF NOT EXISTS organizations (
--     id VARCHAR(36) PRIMARY KEY,
--     organization_code VARCHAR(10) UNIQUE NOT NULL,
--     organization_name VARCHAR(255) NOT NULL,
--     company_name VARCHAR(255) NOT NULL,
--     address TEXT,
--     city VARCHAR(255),
--     province VARCHAR(255),
--     phone VARCHAR(50),
--     email VARCHAR(255),
--     user_id VARCHAR(36) NOT NULL,
--     created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
--     updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
--     FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
--     INDEX idx_organizations_code (organization_code),
--     INDEX idx_organizations_user_id (user_id)
-- );

