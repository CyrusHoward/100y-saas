-- Sample data for 100y-saas demonstration
-- Run this against your SQLite database to populate with demo data

-- Demo users
INSERT OR IGNORE INTO users (id, email, password_hash, created_at, is_active) VALUES 
(1, 'demo@example.com', 'a665a45920422f9d417e4867efdc4fb8a04a1f3fff1fa07e998e86f7f7a27ae3', '2024-01-01 00:00:00', 1),  -- password: hello
(2, 'admin@example.com', '8c6976e5b5410415bde908bd4dee15dfb167a9c873fc4bb8a81f6f2ab448a918', '2024-01-01 00:00:00', 1), -- password: admin
(3, 'user@example.com', 'ef92b778bafe771e89245b89ecbc08a44a4e166c06659911881f383d4473e94f', '2024-01-02 00:00:00', 1),   -- password: secret
(4, 'test@company.com', 'e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855', '2024-01-03 00:00:00', 1), -- password: (empty for demo)
(5, 'manager@startup.io', '2bb80d537b1da3e38bd30361aa855686bde0eacd7162fef6a25fe97bf527a25b', '2024-01-05 00:00:00', 1); -- password: demo123

-- Demo tenants/workspaces
INSERT OR IGNORE INTO tenants (id, name, owner_id, created_at, is_active) VALUES
(1, 'Acme Corporation', 1, '2024-01-01 00:00:00', 1),
(2, 'Tech Startup Inc', 2, '2024-01-01 00:00:00', 1),
(3, 'Freelancer Workspace', 3, '2024-01-02 00:00:00', 1),
(4, 'Demo Company', 4, '2024-01-03 00:00:00', 1);

-- Tenant user associations
INSERT OR IGNORE INTO tenant_users (tenant_id, user_id, role, joined_at) VALUES
-- Acme Corporation
(1, 1, 'owner', '2024-01-01 00:00:00'),
(1, 2, 'admin', '2024-01-01 01:00:00'),
(1, 3, 'member', '2024-01-02 00:00:00'),
-- Tech Startup Inc
(2, 2, 'owner', '2024-01-01 00:00:00'),
(2, 5, 'member', '2024-01-05 00:00:00'),
-- Freelancer Workspace
(3, 3, 'owner', '2024-01-02 00:00:00'),
-- Demo Company
(4, 4, 'owner', '2024-01-03 00:00:00');

-- Subscriptions (demo plans)
INSERT OR IGNORE INTO subscriptions (id, tenant_id, plan, status, starts_at, ends_at, max_items, max_users) VALUES
(1, 1, 'pro', 'active', '2024-01-01 00:00:00', NULL, 1000, 25),
(2, 2, 'starter', 'active', '2024-01-01 00:00:00', '2024-12-31 23:59:59', 500, 10),
(3, 3, 'free', 'active', '2024-01-02 00:00:00', NULL, 100, 5),
(4, 4, 'free', 'active', '2024-01-03 00:00:00', NULL, 100, 5);

-- Sample items/tasks
INSERT OR IGNORE INTO items (id, tenant_id, title, note, created_by, created_at) VALUES
-- Acme Corporation items
(1, 1, 'Launch marketing campaign', 'Q1 product launch with social media focus', 1, '2024-01-05 09:00:00'),
(2, 1, 'Update website design', 'Modernize UI/UX based on user feedback', 2, '2024-01-05 10:30:00'),
(3, 1, 'Hire senior developer', 'Full-stack developer with React/Node.js experience', 1, '2024-01-06 14:15:00'),
(4, 1, 'Security audit', 'Annual penetration testing and vulnerability assessment', 3, '2024-01-07 11:20:00'),
(5, 1, 'Customer feedback analysis', 'Review and categorize Q4 2023 customer surveys', 2, '2024-01-08 16:45:00'),
(6, 1, 'Database optimization', 'Improve query performance for reporting dashboard', 3, '2024-01-09 08:30:00'),

-- Tech Startup Inc items
(7, 2, 'MVP development', 'Build minimum viable product for beta testing', 2, '2024-01-10 09:00:00'),
(8, 2, 'Investor pitch deck', 'Prepare presentation for Series A funding round', 2, '2024-01-10 13:30:00'),
(9, 2, 'User testing sessions', 'Conduct usability tests with 20 target users', 5, '2024-01-11 10:00:00'),
(10, 2, 'API documentation', 'Write comprehensive API docs for developers', 5, '2024-01-12 15:20:00'),

-- Freelancer Workspace items
(11, 3, 'Client website redesign', 'Modernize local restaurant website', 3, '2024-01-15 10:00:00'),
(12, 3, 'Logo design project', 'Create brand identity for tech startup', 3, '2024-01-16 14:30:00'),
(13, 3, 'Email newsletter template', 'Design responsive template for monthly newsletter', 3, '2024-01-17 09:15:00'),

-- Demo Company items  
(14, 4, 'Team onboarding process', 'Streamline new employee orientation', 4, '2024-01-20 11:00:00'),
(15, 4, 'Quarterly business review', 'Analyze performance metrics and set Q2 goals', 4, '2024-01-21 16:30:00');

-- Sample usage events (analytics data)
INSERT OR IGNORE INTO usage_events (tenant_id, user_id, event_type, event_data, created_at) VALUES
-- Recent activity for Acme Corporation
(1, 1, 'item_created', '{"item_id": 1, "title": "Launch marketing campaign"}', '2024-01-05 09:00:00'),
(1, 2, 'item_created', '{"item_id": 2, "title": "Update website design"}', '2024-01-05 10:30:00'),
(1, 3, 'user_login', '{"ip": "192.168.1.100"}', '2024-01-06 08:15:00'),
(1, 1, 'item_created', '{"item_id": 3, "title": "Hire senior developer"}', '2024-01-06 14:15:00'),
(1, 2, 'item_viewed', '{"item_id": 1}', '2024-01-06 16:20:00'),
(1, 3, 'item_created', '{"item_id": 4, "title": "Security audit"}', '2024-01-07 11:20:00'),
(1, 1, 'export_data', '{"format": "csv", "items_count": 4}', '2024-01-07 15:30:00'),

-- Tech Startup activity
(2, 2, 'item_created', '{"item_id": 7, "title": "MVP development"}', '2024-01-10 09:00:00'),
(2, 2, 'item_created', '{"item_id": 8, "title": "Investor pitch deck"}', '2024-01-10 13:30:00'),
(2, 5, 'user_invited', '{"invited_email": "developer@startup.io"}', '2024-01-11 09:00:00'),
(2, 5, 'item_created', '{"item_id": 9, "title": "User testing sessions"}', '2024-01-11 10:00:00'),

-- Freelancer activity
(3, 3, 'item_created', '{"item_id": 11, "title": "Client website redesign"}', '2024-01-15 10:00:00'),
(3, 3, 'subscription_viewed', '{"current_plan": "free"}', '2024-01-15 11:30:00'),
(3, 3, 'item_created', '{"item_id": 12, "title": "Logo design project"}', '2024-01-16 14:30:00');

-- Sample background jobs (some completed, some pending)
INSERT OR IGNORE INTO jobs (id, type, payload, status, attempts, scheduled_at, started_at, completed_at) VALUES
(1, 'cleanup_sessions', '{}', 'completed', 1, '2024-01-01 02:15:00', '2024-01-01 02:15:01', '2024-01-01 02:15:02'),
(2, 'cleanup_usage_events', '{}', 'completed', 1, '2024-01-01 02:15:00', '2024-01-01 02:15:03', '2024-01-01 02:15:05'),
(3, 'send_welcome_email', '{"user_email": "test@company.com", "user_name": "Test User"}', 'completed', 1, '2024-01-03 00:00:00', '2024-01-03 00:00:01', '2024-01-03 00:00:02'),
(4, 'cleanup_sessions', '{}', 'pending', 0, datetime('now', '+1 hour'), NULL, NULL),
(5, 'usage_summary_email', '{"tenant_id": 1, "period": "weekly"}', 'pending', 0, datetime('now', '+2 hours'), NULL, NULL);

-- Sample metadata
INSERT OR IGNORE INTO meta (key, value) VALUES 
('sample_data_loaded', 'true'),
('sample_data_version', '1.0'),
('last_demo_reset', datetime('now'));

-- Summary of loaded data:
-- Users: 5 demo users with simple passwords
-- Tenants: 4 different workspaces/companies  
-- Items: 15 sample tasks/projects across different tenants
-- Analytics: Usage events showing realistic user activity
-- Jobs: Mix of completed and pending background jobs
-- Subscriptions: Different plan types (free, starter, pro)
