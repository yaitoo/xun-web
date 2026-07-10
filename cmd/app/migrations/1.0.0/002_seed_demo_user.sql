-- Seed demo user, password is demo123.
-- Hash = SHA256(demo123) as lowercase hex.
-- Production should use bcrypt or argon2 instead of plain SHA256.
INSERT INTO users (name, email, password_hash, created_at) VALUES (
    'Demo User',
    'demo@example.com',
    'd3ad9315b7be5dd53b31a273b3b3aba5defe700808305aa16a3062b76658a791',
    datetime('now')
);