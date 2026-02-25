CREATE TABLE IF NOT EXISTS customer_accounts (
    id UUID PRIMARY KEY,
    external_ref TEXT NOT NULL UNIQUE,
    legal_name TEXT NOT NULL,
    display_name TEXT NOT NULL,
    segment TEXT NOT NULL DEFAULT 'default',
    status TEXT NOT NULL CHECK (status IN ('prospect', 'active', 'suspended', 'archived')),
    parent_customer_id UUID REFERENCES customer_accounts(id),
    open_invoice_balance_minor BIGINT NOT NULL DEFAULT 0,
    has_active_subscriptions BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_customer_accounts_status ON customer_accounts(status);

CREATE TABLE IF NOT EXISTS customer_preferences (
    customer_id UUID PRIMARY KEY REFERENCES customer_accounts(id) ON DELETE CASCADE,
    locale TEXT NOT NULL DEFAULT 'en-US',
    timezone TEXT NOT NULL DEFAULT 'UTC',
    default_currency TEXT NOT NULL DEFAULT 'USD',
    invoice_delivery_channel TEXT NOT NULL DEFAULT 'email',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS customer_contacts (
    id UUID PRIMARY KEY,
    customer_id UUID NOT NULL REFERENCES customer_accounts(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    email TEXT NOT NULL,
    phone TEXT,
    role TEXT NOT NULL,
    is_primary BOOLEAN NOT NULL DEFAULT FALSE,
    verified_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_customer_contacts_customer_id ON customer_contacts(customer_id);
CREATE UNIQUE INDEX IF NOT EXISTS ux_customer_primary_contact ON customer_contacts(customer_id) WHERE is_primary;

CREATE TABLE IF NOT EXISTS customer_users (
    id UUID PRIMARY KEY,
    customer_id UUID NOT NULL REFERENCES customer_accounts(id) ON DELETE CASCADE,
    user_id TEXT NOT NULL,
    role TEXT NOT NULL CHECK (role IN ('owner', 'admin', 'billing', 'viewer')),
    status TEXT NOT NULL CHECK (status IN ('invited', 'active', 'disabled')) DEFAULT 'invited',
    invited_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    accepted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (customer_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_customer_users_customer_id ON customer_users(customer_id);

CREATE TABLE IF NOT EXISTS outbox_events (
    id UUID PRIMARY KEY,
    aggregate_type TEXT NOT NULL,
    aggregate_id UUID NOT NULL,
    event_name TEXT NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_outbox_events_pending ON outbox_events(processed_at) WHERE processed_at IS NULL;
