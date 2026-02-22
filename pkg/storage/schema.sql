-- 1. Users & Organizations (Multi-tenant Foundation)
CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    name TEXT DEFAULT '',
    phone TEXT DEFAULT '',
    company TEXT DEFAULT '',
    plan TEXT DEFAULT 'free', -- 'free', 'starter', 'pro'
    plan_status TEXT DEFAULT 'active', -- 'active', 'canceled', 'expired'
    current_period_end DATETIME,
    stripe_customer_id TEXT,
    stripe_subscription_id TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);


CREATE TABLE IF NOT EXISTS verification_codes (
    email TEXT NOT NULL,
    code TEXT NOT NULL,
    purpose TEXT NOT NULL, -- 'register' or 'reset_password'
    expires_at DATETIME NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(email, purpose)
);

CREATE TABLE IF NOT EXISTS order_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    gateway TEXT NOT NULL,           -- 'stripe' | 'alipay'
    trade_no TEXT UNIQUE NOT NULL,   -- External transaction ID
    amount INTEGER NOT NULL,         -- Amount in cents
    status TEXT NOT NULL,            -- 'pending', 'paid', 'failed'
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    paid_at DATETIME,
    FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
);


-- 2. WatchBot: Competitors & Pages
CREATE TABLE IF NOT EXISTS competitors (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    domain TEXT NOT NULL,
    status TEXT DEFAULT 'active', -- 'active', 'frozen'
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
    target_type TEXT DEFAULT 'email',
    target_id TEXT DEFAULT '',
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
