ALTER TABLE users
  ADD COLUMN IF NOT EXISTS password_hash VARCHAR(255);

CREATE TABLE IF NOT EXISTS notification_logs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  donor_id UUID REFERENCES users(id) NOT NULL,
  donation_id UUID REFERENCES donation_history(id) NOT NULL,
  type VARCHAR(100) NOT NULL,
  sent_at TIMESTAMPTZ,
  UNIQUE (donation_id, type)
);

CREATE INDEX IF NOT EXISTS idx_notification_logs_donor ON notification_logs(donor_id, sent_at DESC);
CREATE INDEX IF NOT EXISTS idx_donation_history_donor_date ON donation_history(donor_id, donation_date DESC, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_hospitals_active_created ON hospitals(is_active, created_at DESC);

UPDATE users
SET password_hash = '$2b$12$xzq4qIoOgNA9mn5sSsjbZurYUWWlZXL.SZaRRcVJET4htZj.MFKXe'
WHERE id::TEXT IN (
  'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb001',
  'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb002',
  'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb003',
  'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb004',
  'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb005',
  'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb006',
  'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb007',
  'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb008',
  'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb009',
  'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb010',
  'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb011',
  'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb012',
  'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb013',
  'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb014',
  'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb015'
)
  AND COALESCE(password_hash, '') = '';
