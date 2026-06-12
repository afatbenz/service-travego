-- Migration: Create wa_contacts table for WhatsApp AI Assistant
-- Created: 2026-06-12
-- Description: Store WhatsApp phone contacts mapped to organizations and roles

-- Create wa_contacts table
CREATE TABLE IF NOT EXISTS wa_contacts (
    id BIGSERIAL PRIMARY KEY,
    phone VARCHAR(20) UNIQUE NOT NULL,
    name VARCHAR(100),
    role VARCHAR(50),
    organization_id BIGINT NOT NULL,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE
);

-- Create index for faster phone lookups
CREATE INDEX IF NOT EXISTS idx_wa_contacts_phone ON wa_contacts(phone);
CREATE INDEX IF NOT EXISTS idx_wa_contacts_organization_id ON wa_contacts(organization_id);
CREATE INDEX IF NOT EXISTS idx_wa_contacts_active ON wa_contacts(is_active);

-- Add comment for documentation
COMMENT ON TABLE wa_contacts IS 'WhatsApp contact registry for WAAI (WhatsApp AI Assistant) module. Maps phone numbers to organizations and user roles.';
COMMENT ON COLUMN wa_contacts.phone IS 'WhatsApp phone number in format: 628123456789 (without country code prefix or @s.whatsapp.net)';
COMMENT ON COLUMN wa_contacts.role IS 'User role within organization (e.g., direktur, admin, operasional)';
COMMENT ON COLUMN wa_contacts.is_active IS 'Whether this contact is active and can use WAAI';
