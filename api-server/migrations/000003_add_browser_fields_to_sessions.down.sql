-- Remove browser-specific columns from the sessions table
ALTER TABLE sessions
DROP COLUMN IF EXISTS browser_id,
DROP COLUMN IF EXISTS browser_type,
DROP COLUMN IF EXISTS cdp_url,
DROP COLUMN IF EXISTS headless,
DROP COLUMN IF EXISTS viewport_w,
DROP COLUMN IF EXISTS viewport_h,
DROP COLUMN IF EXISTS user_agent;