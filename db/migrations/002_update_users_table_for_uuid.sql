-- Update users table to support UUID and new fields
-- For PostgreSQL
ALTER TABLE users 
  ALTER COLUMN id TYPE VARCHAR(36),
  ADD COLUMN IF NOT EXISTS username VARCHAR(255) UNIQUE,
  ADD COLUMN IF NOT EXISTS status INTEGER DEFAULT 2,
  ADD COLUMN IF NOT EXISTS city VARCHAR(255),
  ADD COLUMN IF NOT EXISTS province VARCHAR(255),
  ADD COLUMN IF NOT EXISTS postal_code VARCHAR(20),
  ADD COLUMN IF NOT EXISTS npwp VARCHAR(50),
  ADD COLUMN IF NOT EXISTS gender VARCHAR(1),
  ADD COLUMN IF NOT EXISTS date_of_birth TIMESTAMP;

-- For MySQL (use this if using MySQL)
-- ALTER TABLE users 
--   MODIFY COLUMN id VARCHAR(36),
--   ADD COLUMN username VARCHAR(255) UNIQUE,
--   ADD COLUMN status INT DEFAULT 2,
--   ADD COLUMN city VARCHAR(255),
--   ADD COLUMN province VARCHAR(255),
--   ADD COLUMN postal_code VARCHAR(20),
--   ADD COLUMN npwp VARCHAR(50),
--   ADD COLUMN gender VARCHAR(1),
--   ADD COLUMN date_of_birth DATETIME;

-- Create index on username
CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
CREATE INDEX IF NOT EXISTS idx_users_status ON users(status);

