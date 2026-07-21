CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE cooperatives (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    legal_number TEXT,
    address TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cooperative_id UUID NOT NULL REFERENCES cooperatives(id),
    email TEXT UNIQUE,
    phone TEXT UNIQUE,
    password_hash TEXT NOT NULL,
    role TEXT NOT NULL CHECK (role IN ('admin', 'farmer')),
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'invited', 'disabled')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id),
    cooperative_id UUID NOT NULL REFERENCES cooperatives(id),
    member_number TEXT NOT NULL,
    nik TEXT,
    full_name TEXT NOT NULL,
    address TEXT,
    land_area_ha NUMERIC(10, 2),
    bank_name TEXT,
    bank_account_no TEXT,
    join_date DATE,
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (cooperative_id, member_number)
);

CREATE TABLE price_references (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cooperative_id UUID NOT NULL REFERENCES cooperatives(id),
    grade TEXT NOT NULL CHECK (grade IN ('A', 'B', 'C')),
    price_per_kg NUMERIC(12, 2) NOT NULL,
    effective_date DATE NOT NULL,
    set_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE harvest_records (
    id UUID PRIMARY KEY,
    member_id UUID NOT NULL REFERENCES members(id),
    cooperative_id UUID NOT NULL REFERENCES cooperatives(id),
    harvest_date DATE NOT NULL,
    tbs_weight_kg NUMERIC(10, 2) NOT NULL CHECK (tbs_weight_kg > 0),
    quality_grade TEXT NOT NULL CHECK (quality_grade IN ('A', 'B', 'C')),
    gps_lat DOUBLE PRECISION,
    gps_lng DOUBLE PRECISION,
    photo_url TEXT,
    device_id TEXT,
    client_created_at TIMESTAMPTZ NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending_review' CHECK (status IN ('pending_review', 'validated', 'rejected', 'voided')),
    validated_by UUID REFERENCES users(id),
    validated_at TIMESTAMPTZ,
    price_id UUID REFERENCES price_references(id),
    payout_run_item_id UUID,
    corrects_id UUID REFERENCES harvest_records(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_harvest_records_member ON harvest_records(member_id);
CREATE INDEX idx_harvest_records_status ON harvest_records(cooperative_id, status);

CREATE TABLE deductions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    member_id UUID NOT NULL REFERENCES members(id),
    type TEXT NOT NULL CHECK (type IN ('loan', 'input_purchase', 'cooperative_fee')),
    principal_amount NUMERIC(12, 2) NOT NULL,
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'closed')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE deduction_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    deduction_id UUID NOT NULL REFERENCES deductions(id),
    payout_run_item_id UUID,
    amount_applied NUMERIC(12, 2) NOT NULL,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE payout_runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cooperative_id UUID NOT NULL REFERENCES cooperatives(id),
    period_start DATE NOT NULL,
    period_end DATE NOT NULL,
    status TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'calculating', 'calculated', 'approved', 'paid')),
    total_amount NUMERIC(14, 2) NOT NULL DEFAULT 0,
    created_by UUID NOT NULL REFERENCES users(id),
    approved_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE payout_run_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    payout_run_id UUID NOT NULL REFERENCES payout_runs(id),
    member_id UUID NOT NULL REFERENCES members(id),
    gross_amount NUMERIC(14, 2) NOT NULL DEFAULT 0,
    deductions_total NUMERIC(14, 2) NOT NULL DEFAULT 0,
    net_amount NUMERIC(14, 2) NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'paid'))
);

ALTER TABLE harvest_records
    ADD CONSTRAINT fk_harvest_records_payout_run_item
    FOREIGN KEY (payout_run_item_id) REFERENCES payout_run_items(id);

ALTER TABLE deduction_transactions
    ADD CONSTRAINT fk_deduction_transactions_payout_run_item
    FOREIGN KEY (payout_run_item_id) REFERENCES payout_run_items(id);

CREATE TABLE notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    type TEXT NOT NULL,
    title TEXT NOT NULL,
    body TEXT,
    related_entity_type TEXT,
    related_entity_id UUID,
    read_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE sync_batches (
    id UUID PRIMARY KEY,
    member_id UUID NOT NULL REFERENCES members(id),
    device_id TEXT,
    submitted_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    record_count INTEGER NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'processed'
);

CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_id UUID REFERENCES users(id),
    action TEXT NOT NULL,
    entity_type TEXT NOT NULL,
    entity_id UUID,
    before JSONB,
    after JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
