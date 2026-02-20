-- 1. Users & Organizations (Multi-tenant Foundation)
CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    plan TEXT DEFAULT 'free', -- 'free', 'starter', 'pro'
    stripe_customer_id TEXT,
    stripe_subscription_id TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 2. WatchBot: Competitors & Pages
CREATE TABLE IF NOT EXISTS competitors (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    domain TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE(user_id, domain)
);

CREATE TABLE IF NOT EXISTS alert_rules (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    competitor_id INTEGER, -- Optional: if NULL, applies to all competitors
    rule_type TEXT NOT NULL, -- 'severity', 'keyword'
    rule_value TEXT NOT NULL, -- 'high', 'pricing'
    action TEXT NOT NULL, -- 'email', 'webhook'
    is_active BOOLEAN DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY(competitor_id) REFERENCES competitors(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS pages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    competitor_id INTEGER NOT NULL,
    url TEXT NOT NULL,
    page_type TEXT NOT NULL, -- 'pricing', 'features', 'changelog'
    last_checked_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(competitor_id) REFERENCES competitors(id) ON DELETE CASCADE
);

-- 3. WatchBot: Snapshots & Analyses
CREATE TABLE IF NOT EXISTS snapshots (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    page_id INTEGER NOT NULL,
    checksum TEXT NOT NULL,
    content TEXT NOT NULL,
    captured_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(page_id) REFERENCES pages(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS analyses (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    page_id INTEGER NOT NULL,
    old_snapshot_id INTEGER,
    new_snapshot_id INTEGER NOT NULL,
    severity TEXT, -- 'critical', 'important', 'minor', 'none'
    summary TEXT,
    raw_diff TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(page_id) REFERENCES pages(id) ON DELETE CASCADE,
    FOREIGN KEY(old_snapshot_id) REFERENCES snapshots(id) ON DELETE SET NULL,
    FOREIGN KEY(new_snapshot_id) REFERENCES snapshots(id) ON DELETE CASCADE
);

-- 4. NewsBot: Global Digestion
CREATE TABLE IF NOT EXISTS news_articles (
    id TEXT PRIMARY KEY, -- SHA256 of URL
    url TEXT UNIQUE NOT NULL,
    title TEXT NOT NULL,
    source TEXT NOT NULL,
    summary TEXT,
    published_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS news_digests (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    date TEXT UNIQUE NOT NULL, -- e.g. "2026-02-20"
    content_json TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
