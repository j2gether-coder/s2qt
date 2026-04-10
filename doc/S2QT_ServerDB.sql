-- PostgreSQL용
CREATE TABLE users (
    id                  BIGSERIAL PRIMARY KEY,
    email               TEXT NOT NULL UNIQUE,
    display_name        TEXT,
    status              TEXT NOT NULL DEFAULT 'active', -- active / suspended / deleted
    created_at          TIMESTAMP NOT NULL DEFAULT now(),
    updated_at          TIMESTAMP NOT NULL DEFAULT now(),
    last_login_at       TIMESTAMP
);

CREATE INDEX idx_users_status ON users(status);

CREATE TABLE auth_accounts (
    id                      BIGSERIAL PRIMARY KEY,
    user_id                 BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    login_id                TEXT NOT NULL UNIQUE, -- 보통 email과 같게 시작 가능
    password_hash           TEXT NOT NULL,
    password_algo           TEXT NOT NULL DEFAULT 'argon2id',
    email_verified_at       TIMESTAMP,
    password_changed_at     TIMESTAMP,
    failed_login_count      INT NOT NULL DEFAULT 0,
    locked_until            TIMESTAMP,
    created_at              TIMESTAMP NOT NULL DEFAULT now(),
    updated_at              TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX idx_auth_accounts_user_id ON auth_accounts(user_id);
CREATE INDEX idx_auth_accounts_login_id ON auth_accounts(login_id);

CREATE TABLE auth_sessions (
    id                      BIGSERIAL PRIMARY KEY,
    user_id                 BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    session_token_hash      TEXT NOT NULL UNIQUE,
    refresh_token_hash      TEXT,
    user_agent              TEXT,
    ip_masked               TEXT,
    created_at              TIMESTAMP NOT NULL DEFAULT now(),
    expires_at              TIMESTAMP NOT NULL,
    revoked_at              TIMESTAMP
);

CREATE INDEX idx_auth_sessions_user_id ON auth_sessions(user_id);
CREATE INDEX idx_auth_sessions_expires_at ON auth_sessions(expires_at);
CREATE INDEX idx_auth_sessions_revoked_at ON auth_sessions(revoked_at);

CREATE TABLE auth_email_tokens (
    id                      BIGSERIAL PRIMARY KEY,
    user_id                 BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_type              TEXT NOT NULL, -- verify_email / reset_password
    token_hash              TEXT NOT NULL UNIQUE,
    expires_at              TIMESTAMP NOT NULL,
    used_at                 TIMESTAMP,
    created_at              TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX idx_auth_email_tokens_user_id ON auth_email_tokens(user_id);
CREATE INDEX idx_auth_email_tokens_type ON auth_email_tokens(token_type);
CREATE INDEX idx_auth_email_tokens_expires_at ON auth_email_tokens(expires_at);

CREATE TABLE subscriptions (
    id                      BIGSERIAL PRIMARY KEY,
    user_id                 BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    subscription_type       TEXT NOT NULL, -- monthly / yearly / trial
    status                  TEXT NOT NULL, -- active / expired / canceled / trial
    start_at                TIMESTAMP NOT NULL,
    end_at                  TIMESTAMP NOT NULL,
    monthly_quota           INT NOT NULL DEFAULT 5,
    created_at              TIMESTAMP NOT NULL DEFAULT now(),
    updated_at              TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX idx_subscriptions_user_id ON subscriptions(user_id);
CREATE INDEX idx_subscriptions_status ON subscriptions(status);
CREATE INDEX idx_subscriptions_end_at ON subscriptions(end_at);

CREATE TABLE subscriptions (
    id                      BIGSERIAL PRIMARY KEY,
    user_id                 BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    subscription_type       TEXT NOT NULL, -- monthly / yearly / trial
    status                  TEXT NOT NULL, -- active / expired / canceled / trial
    start_at                TIMESTAMP NOT NULL,
    end_at                  TIMESTAMP NOT NULL,
    monthly_quota           INT NOT NULL DEFAULT 5,
    created_at              TIMESTAMP NOT NULL DEFAULT now(),
    updated_at              TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX idx_subscriptions_user_id ON subscriptions(user_id);
CREATE INDEX idx_subscriptions_status ON subscriptions(status);
CREATE INDEX idx_subscriptions_end_at ON subscriptions(end_at);

CREATE TABLE qt_documents (
    id                      BIGSERIAL PRIMARY KEY,
    user_id                 BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    title                   TEXT NOT NULL,
    preacher                TEXT NOT NULL,
    tradition               TEXT NOT NULL DEFAULT 'bible',
    source_type             TEXT NOT NULL, -- youtube / audio / text
    generation_mode         TEXT NOT NULL DEFAULT 'subscription',

    primary_ref_text        TEXT,
    qt_markdown             TEXT NOT NULL,
    qt_summary              TEXT,
    has_hymn_recommend      BOOLEAN NOT NULL DEFAULT false,

    is_deleted              BOOLEAN NOT NULL DEFAULT false,
    created_at              TIMESTAMP NOT NULL DEFAULT now(),
    updated_at              TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX idx_qt_documents_user_id ON qt_documents(user_id);
CREATE INDEX idx_qt_documents_title ON qt_documents(title);
CREATE INDEX idx_qt_documents_preacher ON qt_documents(preacher);
CREATE INDEX idx_qt_documents_tradition ON qt_documents(tradition);
CREATE INDEX idx_qt_documents_created_at ON qt_documents(created_at);

CREATE TABLE qt_scripture_refs (
    id                      BIGSERIAL PRIMARY KEY,
    qt_document_id          BIGINT NOT NULL REFERENCES qt_documents(id) ON DELETE CASCADE,
    ref_role                TEXT NOT NULL, -- primary / mentioned
    tradition               TEXT NOT NULL,
    work_id                 TEXT NOT NULL,

    ref_start_1             INT,
    ref_start_2             INT,
    ref_start_3             INT,
    ref_end_1               INT,
    ref_end_2               INT,
    ref_end_3               INT,

    display_text            TEXT NOT NULL,
    sort_order              INT NOT NULL DEFAULT 0
);

CREATE INDEX idx_qt_scripture_refs_doc_id ON qt_scripture_refs(qt_document_id);
CREATE INDEX idx_qt_scripture_refs_lookup
    ON qt_scripture_refs(tradition, work_id, ref_start_1, ref_start_2, ref_start_3);
CREATE INDEX idx_qt_scripture_refs_role ON qt_scripture_refs(ref_role);

CREATE TABLE templates (
    id                      BIGSERIAL PRIMARY KEY,
    template_code           TEXT NOT NULL UNIQUE,
    template_name           TEXT NOT NULL,
    template_type           TEXT NOT NULL, -- free_pdf / paid_pdf / booklet / annual_ebook / view
    tradition_scope         TEXT,
    is_active               BOOLEAN NOT NULL DEFAULT true,
    is_paid                 BOOLEAN NOT NULL DEFAULT false,
    price_krw               INT NOT NULL DEFAULT 0,
    created_at              TIMESTAMP NOT NULL DEFAULT now(),
    updated_at              TIMESTAMP NOT NULL DEFAULT now()
);

CREATE TABLE template_purchases (
    id                      BIGSERIAL PRIMARY KEY,
    user_id                 BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    template_code           TEXT NOT NULL,
    product_type            TEXT NOT NULL, -- single_pdf / booklet / annual_ebook
    price_krw               INT NOT NULL,
    purchased_at            TIMESTAMP NOT NULL DEFAULT now(),
    expires_at              TIMESTAMP
);

CREATE INDEX idx_template_purchases_user_id ON template_purchases(user_id);
CREATE INDEX idx_template_purchases_template_code ON template_purchases(template_code);

CREATE TABLE billing_transactions (
    id                      BIGSERIAL PRIMARY KEY,
    user_id                 BIGINT REFERENCES users(id) ON DELETE SET NULL,
    transaction_type        TEXT NOT NULL, -- subscription / template_purchase
    provider                TEXT NOT NULL, -- toss / inicis / etc
    provider_tx_id          TEXT,
    item_code               TEXT NOT NULL, -- monthly_9900 / yearly_99000 / tpl_500 ...
    amount_krw              INT NOT NULL,
    payment_fee_krw         INT,
    status                  TEXT NOT NULL, -- paid / canceled / failed / refunded
    paid_at                 TIMESTAMP,
    created_at              TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX idx_billing_transactions_user_id ON billing_transactions(user_id);
CREATE INDEX idx_billing_transactions_status ON billing_transactions(status);
CREATE INDEX idx_billing_transactions_provider_tx_id ON billing_transactions(provider_tx_id);

CREATE TABLE audit_logs (
    id                      BIGSERIAL PRIMARY KEY,
    user_id                 BIGINT REFERENCES users(id) ON DELETE SET NULL,
    event_type              TEXT NOT NULL, -- login_success / login_failed / password_changed / subscription_started ...
    event_detail            TEXT,
    ip_masked               TEXT,
    user_agent              TEXT,
    created_at              TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_event_type ON audit_logs(event_type);
CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at);
