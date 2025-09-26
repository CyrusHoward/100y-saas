PRAGMA foreign_keys = ON;
CREATE TABLE IF NOT EXISTS meta (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL
);

-- Users and Authentication
CREATE TABLE IF NOT EXISTS users (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  email TEXT UNIQUE NOT NULL,
  password_hash TEXT NOT NULL,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  last_login DATETIME,
  is_active BOOLEAN DEFAULT 1
);

CREATE TABLE IF NOT EXISTS sessions (
  token TEXT PRIMARY KEY,
  user_id INTEGER NOT NULL REFERENCES users(id),
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  expires_at DATETIME NOT NULL
);

-- Multi-tenancy
CREATE TABLE IF NOT EXISTS tenants (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  owner_id INTEGER NOT NULL REFERENCES users(id),
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  is_active BOOLEAN DEFAULT 1
);

CREATE TABLE IF NOT EXISTS tenant_users (
  tenant_id INTEGER NOT NULL REFERENCES tenants(id),
  user_id INTEGER NOT NULL REFERENCES users(id),
  role TEXT NOT NULL DEFAULT 'member',
  joined_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (tenant_id, user_id)
);

-- Subscriptions (tracking only, no payment processing)
CREATE TABLE IF NOT EXISTS subscriptions (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  tenant_id INTEGER NOT NULL REFERENCES tenants(id),
  plan TEXT NOT NULL DEFAULT 'free',
  status TEXT NOT NULL DEFAULT 'active',
  starts_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  ends_at DATETIME,
  max_items INTEGER DEFAULT 100,
  max_users INTEGER DEFAULT 5
);

-- Usage Analytics
CREATE TABLE IF NOT EXISTS usage_events (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  tenant_id INTEGER REFERENCES tenants(id),
  user_id INTEGER REFERENCES users(id),
  event_type TEXT NOT NULL,
  event_data TEXT, -- JSON
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_usage_events_tenant_date ON usage_events(tenant_id, created_at);
CREATE INDEX IF NOT EXISTS idx_sessions_expires ON sessions(expires_at);

-- Background Jobs
CREATE TABLE IF NOT EXISTS jobs (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  type TEXT NOT NULL,
  payload TEXT NOT NULL, -- JSON
  status TEXT NOT NULL DEFAULT 'pending',
  attempts INTEGER DEFAULT 0,
  max_attempts INTEGER DEFAULT 3,
  scheduled_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  started_at DATETIME,
  completed_at DATETIME,
  error TEXT
);

CREATE INDEX IF NOT EXISTS idx_jobs_status_scheduled ON jobs(status, scheduled_at);

-- Updated Items table with multi-tenancy
CREATE TABLE IF NOT EXISTS items (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  tenant_id INTEGER NOT NULL REFERENCES tenants(id),
  title TEXT NOT NULL,
  note TEXT DEFAULT '',
  created_by INTEGER REFERENCES users(id),
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_items_tenant ON items(tenant_id);

INSERT OR IGNORE INTO meta(key,value) VALUES ('schema_version','2');
