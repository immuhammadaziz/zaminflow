-- +goose Up
CREATE TABLE regions (
    code TEXT PRIMARY KEY,
    name_uz TEXT NOT NULL,
    name_ru TEXT NOT NULL,
    name_en TEXT NOT NULL,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO regions (code, name_uz, name_ru, name_en) VALUES
    ('andijan', 'Andijon', 'Андижанская область', 'Andijan Region'),
    ('bukhara', 'Buxoro', 'Бухарская область', 'Bukhara Region'),
    ('fergana', 'Fargʻona', 'Ферганская область', 'Fergana Region'),
    ('jizzakh', 'Jizzax', 'Джизакская область', 'Jizzakh Region'),
    ('kashkadarya', 'Qashqadaryo', 'Кашкадарьинская область', 'Kashkadarya Region'),
    ('khorezm', 'Xorazm', 'Хорезмская область', 'Khorezm Region'),
    ('namangan', 'Namangan', 'Наманганская область', 'Namangan Region'),
    ('navoi', 'Navoiy', 'Навоийская область', 'Navoiy Region'),
    ('republic_of_karakalpakstan', 'Qoraqalpogʻiston Respublikasi', 'Республика Каракалпакстан', 'Republic of Karakalpakstan'),
    ('samarkand', 'Samarqand', 'Самаркандская область', 'Samarkand Region'),
    ('sirdarya', 'Sirdaryo', 'Сырдарьинская область', 'Sirdaryo Region'),
    ('surkhandarya', 'Surxondaryo', 'Сурхандарьинская область', 'Surkhandarya Region'),
    ('tashkent_city', 'Toshkent shahri', 'город Ташкент', 'Tashkent City'),
    ('tashkent_region', 'Toshkent viloyati', 'Ташкентская область', 'Tashkent Region');

CREATE TABLE users (
    id UUID PRIMARY KEY,
    role TEXT NOT NULL CHECK (role IN ('farmer', 'processor', 'admin')),
    account_status TEXT NOT NULL CHECK (account_status IN ('pending', 'approved', 'rejected', 'suspended')),
    email CITEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    email_verified_at TIMESTAMPTZ,
    full_name TEXT NOT NULL,
    phone TEXT,
    organization_name TEXT,
    region_code TEXT REFERENCES regions(code) ON DELETE RESTRICT,
    stir TEXT,
    rejection_reason TEXT,
    admin_mfa_required BOOLEAN NOT NULL DEFAULT FALSE,
    admin_mfa_enabled_at TIMESTAMPTZ,
    admin_totp_secret BYTEA,
    failed_login_attempts INTEGER NOT NULL DEFAULT 0 CHECK (failed_login_attempts >= 0),
    locked_until TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (
        (role = 'admin' AND account_status = 'approved')
        OR role <> 'admin'
    ),
    CHECK (
        role = 'admin'
        OR (phone IS NOT NULL AND organization_name IS NOT NULL AND region_code IS NOT NULL AND stir IS NOT NULL)
    )
);
CREATE INDEX users_status_role_idx ON users (account_status, role);
CREATE INDEX users_region_idx ON users (region_code);
CREATE INDEX users_created_at_idx ON users (created_at DESC);

CREATE TABLE account_reviews (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    reviewer_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    decision TEXT NOT NULL CHECK (decision IN ('approved', 'rejected', 'suspended', 'reinstated', 'resubmitted')),
    reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX account_reviews_user_created_idx ON account_reviews (user_id, created_at DESC);
CREATE INDEX account_reviews_reviewer_idx ON account_reviews (reviewer_id);
CREATE INDEX account_reviews_decision_created_idx ON account_reviews (decision, created_at DESC);

CREATE TABLE account_deletion_requests (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    status TEXT NOT NULL DEFAULT 'requested' CHECK (status IN ('requested', 'completed', 'rejected')),
    requested_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at TIMESTAMPTZ,
    reason TEXT
);
CREATE INDEX account_deletion_user_status_idx ON account_deletion_requests (user_id, status);
CREATE INDEX account_deletion_requested_at_idx ON account_deletion_requests (requested_at DESC);

CREATE TABLE sessions (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    family_id UUID NOT NULL,
    refresh_token_hash BYTEA NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL,
    last_used_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    revoked_at TIMESTAMPTZ,
    replaced_by UUID REFERENCES sessions(id) ON DELETE SET NULL,
    user_agent TEXT,
    ip_address INET
);
CREATE INDEX sessions_user_revoked_expires_idx ON sessions (user_id, revoked_at, expires_at);
CREATE INDEX sessions_family_idx ON sessions (family_id);
CREATE INDEX sessions_replaced_by_idx ON sessions (replaced_by);

CREATE TABLE one_time_tokens (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash BYTEA NOT NULL UNIQUE,
    purpose TEXT NOT NULL CHECK (purpose IN ('email_verification', 'password_reset')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ
);
CREATE INDEX one_time_tokens_user_purpose_idx ON one_time_tokens (user_id, purpose);
CREATE INDEX one_time_tokens_expires_idx ON one_time_tokens (expires_at);

CREATE TABLE admin_recovery_codes (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    code_hash BYTEA NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    used_at TIMESTAMPTZ
);
CREATE INDEX admin_recovery_codes_user_used_idx ON admin_recovery_codes (user_id, used_at);

CREATE TABLE audit_events (
    id UUID PRIMARY KEY,
    actor_id UUID REFERENCES users(id) ON DELETE SET NULL,
    action TEXT NOT NULL,
    target_type TEXT NOT NULL,
    target_id UUID,
    request_id TEXT NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX audit_events_actor_created_idx ON audit_events (actor_id, created_at DESC);
CREATE INDEX audit_events_action_created_idx ON audit_events (action, created_at DESC);
CREATE INDEX audit_events_target_idx ON audit_events (target_type, target_id);
CREATE INDEX audit_events_created_at_idx ON audit_events (created_at DESC);

CREATE TABLE mail_outbox (
    id UUID PRIMARY KEY,
    recipient TEXT NOT NULL,
    subject TEXT NOT NULL,
    body TEXT NOT NULL,
    attempts INTEGER NOT NULL DEFAULT 0 CHECK (attempts >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    sent_at TIMESTAMPTZ,
    failed_at TIMESTAMPTZ,
    last_error TEXT
);
CREATE INDEX mail_outbox_pending_idx ON mail_outbox (next_attempt_at, created_at)
    WHERE sent_at IS NULL AND failed_at IS NULL;

CREATE TABLE request_rate_limits (
    bucket_key TEXT PRIMARY KEY,
    window_started_at TIMESTAMPTZ NOT NULL,
    request_count INTEGER NOT NULL CHECK (request_count > 0),
    blocked_until TIMESTAMPTZ
);
CREATE INDEX request_rate_limits_window_idx ON request_rate_limits (window_started_at);

CREATE TABLE milk_listings (
    id UUID PRIMARY KEY,
    farmer_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    region_code TEXT NOT NULL REFERENCES regions(code) ON DELETE RESTRICT,
    available_liters BIGINT NOT NULL CHECK (available_liters >= 0),
    reserved_liters BIGINT NOT NULL DEFAULT 0 CHECK (reserved_liters >= 0),
    fat_percent NUMERIC(5,2) NOT NULL CHECK (fat_percent >= 0 AND fat_percent <= 100),
    price_per_liter_uzs BIGINT NOT NULL CHECK (price_per_liter_uzs > 0),
    available_from TIMESTAMPTZ NOT NULL,
    available_until TIMESTAMPTZ NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('draft', 'active', 'paused', 'sold_out', 'expired', 'archived')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (reserved_liters <= available_liters),
    CHECK (available_until > available_from)
);
CREATE INDEX milk_listings_farmer_status_idx ON milk_listings (farmer_id, status);
CREATE INDEX milk_listings_region_status_idx ON milk_listings (region_code, status);
CREATE INDEX milk_listings_status_availability_idx ON milk_listings (status, available_from, available_until);

CREATE TABLE orders (
    id UUID PRIMARY KEY,
    listing_id UUID NOT NULL REFERENCES milk_listings(id) ON DELETE RESTRICT,
    farmer_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    processor_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    liters BIGINT NOT NULL CHECK (liters > 0),
    price_per_liter_uzs BIGINT NOT NULL CHECK (price_per_liter_uzs > 0),
    total_uzs BIGINT NOT NULL CHECK (total_uzs > 0),
    status TEXT NOT NULL CHECK (status IN ('placed', 'accepted', 'rejected', 'cancelled', 'ready_for_pickup', 'picked_up', 'delivered', 'completed', 'disputed')),
    expected_pickup_at TIMESTAMPTZ NOT NULL,
    note TEXT,
    quality_outcome TEXT CHECK (quality_outcome IS NULL OR quality_outcome IN ('accepted', 'rejected')),
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (total_uzs = liters * price_per_liter_uzs),
    CHECK (farmer_id <> processor_id)
);
CREATE INDEX orders_listing_idx ON orders (listing_id);
CREATE INDEX orders_farmer_status_created_idx ON orders (farmer_id, status, created_at DESC);
CREATE INDEX orders_processor_status_created_idx ON orders (processor_id, status, created_at DESC);
CREATE INDEX orders_status_created_idx ON orders (status, created_at DESC);
CREATE INDEX orders_completed_at_idx ON orders (completed_at DESC);
CREATE INDEX orders_pickup_status_idx ON orders (expected_pickup_at, status);

CREATE TABLE order_status_history (
    id UUID PRIMARY KEY,
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE RESTRICT,
    actor_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    old_status TEXT,
    new_status TEXT NOT NULL CHECK (new_status IN ('placed', 'accepted', 'rejected', 'cancelled', 'ready_for_pickup', 'picked_up', 'delivered', 'completed', 'disputed')),
    note TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX order_status_history_order_created_idx ON order_status_history (order_id, created_at);
CREATE INDEX order_status_history_actor_idx ON order_status_history (actor_id);

CREATE TABLE quality_reports (
    id UUID PRIMARY KEY,
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE RESTRICT,
    reporter_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    measurements JSONB NOT NULL,
    outcome TEXT NOT NULL CHECK (outcome IN ('accepted', 'rejected')),
    note TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX quality_reports_order_created_idx ON quality_reports (order_id, created_at DESC);
CREATE INDEX quality_reports_reporter_created_idx ON quality_reports (reporter_id, created_at DESC);
CREATE INDEX quality_reports_outcome_created_idx ON quality_reports (outcome, created_at DESC);

CREATE TABLE order_disputes (
    id UUID PRIMARY KEY,
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE RESTRICT,
    opened_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    reason TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX order_disputes_order_idx ON order_disputes (order_id);
CREATE INDEX order_disputes_created_at_idx ON order_disputes (created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS mail_outbox;
DROP TABLE IF EXISTS request_rate_limits;
DROP TABLE IF EXISTS order_disputes;
DROP TABLE IF EXISTS quality_reports;
DROP TABLE IF EXISTS order_status_history;
DROP TABLE IF EXISTS orders;
DROP TABLE IF EXISTS milk_listings;
DROP TABLE IF EXISTS audit_events;
DROP TABLE IF EXISTS admin_recovery_codes;
DROP TABLE IF EXISTS one_time_tokens;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS account_deletion_requests;
DROP TABLE IF EXISTS account_reviews;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS regions;
