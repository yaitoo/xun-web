-- Seed demo user (password: demo123)
INSERT INTO users (name, email, password_hash, created_at) VALUES (
    'Demo User',
    'demo@example.com',
    'ef92b778bafe771e89245b89ecbc08a44a4e166c06659911881f383d4473e94f',
    datetime('now')
);
