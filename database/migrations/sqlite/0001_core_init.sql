-- 0001_core_init.sql
--
-- EduPilot Core: the institutional concepts every domain depends on.
-- Organisation scoping is structural from the first migration, as required by
-- docs/adr/ADR-009. Money is stored as INTEGER minor units paired with a
-- currency code, never as a float (docs/adr/ADR-005). Primary keys are UUIDv7
-- text, never auto-increment business identifiers (docs/adr/ADR-004).
--
-- Released migrations are immutable. Add a new migration instead of editing
-- this file.

CREATE TABLE organisations (
    id              TEXT    PRIMARY KEY,
    name            TEXT    NOT NULL,
    legal_name      TEXT    NOT NULL DEFAULT '',
    currency        TEXT    NOT NULL,
    country         TEXT    NOT NULL DEFAULT '',
    default_locale  TEXT    NOT NULL DEFAULT 'en',
    tax_number      TEXT    NOT NULL DEFAULT '',
    address_line1   TEXT    NOT NULL DEFAULT '',
    address_line2   TEXT    NOT NULL DEFAULT '',
    city            TEXT    NOT NULL DEFAULT '',
    postal_code     TEXT    NOT NULL DEFAULT '',
    country_code    TEXT    NOT NULL DEFAULT '',
    is_active       INTEGER NOT NULL DEFAULT 1 CHECK (is_active IN (0, 1)),
    version         INTEGER NOT NULL DEFAULT 1,
    created_at      TEXT    NOT NULL,
    updated_at      TEXT    NOT NULL
);

CREATE TABLE sites (
    id               TEXT    PRIMARY KEY,
    organisation_id  TEXT    NOT NULL REFERENCES organisations (id) ON DELETE CASCADE,
    name             TEXT    NOT NULL,
    node_id          TEXT    NOT NULL,
    mode             TEXT    NOT NULL DEFAULT 'standalone'
                     CHECK (mode IN ('standalone', 'site_server', 'cloud')),
    created_at       TEXT    NOT NULL,
    updated_at       TEXT    NOT NULL,
    UNIQUE (organisation_id, node_id)
);

CREATE INDEX idx_sites_organisation ON sites (organisation_id);

CREATE TABLE campuses (
    id               TEXT    PRIMARY KEY,
    organisation_id  TEXT    NOT NULL REFERENCES organisations (id) ON DELETE CASCADE,
    name             TEXT    NOT NULL,
    code             TEXT    NOT NULL,
    address_line1    TEXT    NOT NULL DEFAULT '',
    city             TEXT    NOT NULL DEFAULT '',
    country_code     TEXT    NOT NULL DEFAULT '',
    phone            TEXT    NOT NULL DEFAULT '',
    email            TEXT    NOT NULL DEFAULT '',
    is_primary       INTEGER NOT NULL DEFAULT 0 CHECK (is_primary IN (0, 1)),
    is_active        INTEGER NOT NULL DEFAULT 1 CHECK (is_active IN (0, 1)),
    version          INTEGER NOT NULL DEFAULT 1,
    created_at       TEXT    NOT NULL,
    updated_at       TEXT    NOT NULL,
    UNIQUE (organisation_id, code)
);

CREATE INDEX idx_campuses_organisation ON campuses (organisation_id);

-- At most one primary campus per organisation: a school group has one home
-- campus, and the shell needs a single address to print on official documents.
CREATE UNIQUE INDEX idx_campuses_single_primary ON campuses (organisation_id) WHERE is_primary = 1;

CREATE TABLE academic_years (
    id               TEXT    PRIMARY KEY,
    organisation_id  TEXT    NOT NULL REFERENCES organisations (id) ON DELETE CASCADE,
    name             TEXT    NOT NULL,
    starts_on        TEXT    NOT NULL,
    ends_on          TEXT    NOT NULL,
    is_current       INTEGER NOT NULL DEFAULT 0 CHECK (is_current IN (0, 1)),
    version          INTEGER NOT NULL DEFAULT 1,
    created_at       TEXT    NOT NULL,
    updated_at       TEXT    NOT NULL,
    CHECK (ends_on > starts_on),
    UNIQUE (organisation_id, name)
);

CREATE INDEX idx_academic_years_organisation ON academic_years (organisation_id);

-- Exactly one current year per organisation: the platform's time axis. A school
-- that has not opened its first year yet has none, which the partial index
-- allows, but two current years is always wrong.
CREATE UNIQUE INDEX idx_academic_years_single_current ON academic_years (organisation_id) WHERE is_current = 1;

CREATE TABLE academic_terms (
    id                 TEXT    PRIMARY KEY,
    organisation_id    TEXT    NOT NULL REFERENCES organisations (id) ON DELETE CASCADE,
    academic_year_id   TEXT    NOT NULL REFERENCES academic_years (id) ON DELETE CASCADE,
    name               TEXT    NOT NULL,
    sequence           INTEGER NOT NULL,
    starts_on          TEXT    NOT NULL,
    ends_on            TEXT    NOT NULL,
    created_at         TEXT    NOT NULL,
    updated_at         TEXT    NOT NULL,
    CHECK (ends_on > starts_on),
    UNIQUE (academic_year_id, sequence)
);

CREATE INDEX idx_academic_terms_year ON academic_terms (academic_year_id);

-- Documents are declared before people because a person carries a reference to
-- their photograph, and PostgreSQL resolves a foreign key target at DDL time.
CREATE TABLE documents (
    id                TEXT    PRIMARY KEY,
    organisation_id   TEXT    NOT NULL REFERENCES organisations (id) ON DELETE CASCADE,
    entity_type       TEXT    NOT NULL,
    entity_id         TEXT    NOT NULL,
    file_name         TEXT    NOT NULL,
    content_type      TEXT    NOT NULL,
    size_bytes        INTEGER NOT NULL DEFAULT 0,
    checksum_sha256   TEXT    NOT NULL DEFAULT '',
    storage_path      TEXT    NOT NULL,
    uploaded_by       TEXT    NOT NULL DEFAULT '',
    created_at        TEXT    NOT NULL
);

CREATE INDEX idx_documents_entity ON documents (organisation_id, entity_type, entity_id);

-- A person is an identity, not a role: who somebody is, independent of what
-- they do at the school. Roles live in person_roles, so one person can be a
-- student and an employee and a guardian at the same time
-- (docs/adr/ADR-011).
CREATE TABLE people (
    id                TEXT    PRIMARY KEY,
    organisation_id   TEXT    NOT NULL REFERENCES organisations (id) ON DELETE CASCADE,
    first_name        TEXT    NOT NULL,
    middle_name       TEXT    NOT NULL DEFAULT '',
    last_name         TEXT    NOT NULL,
    preferred_name    TEXT    NOT NULL DEFAULT '',
    email             TEXT    NOT NULL DEFAULT '',
    phone             TEXT    NOT NULL DEFAULT '',
    secondary_phone   TEXT    NOT NULL DEFAULT '',
    date_of_birth     TEXT    NULL,
    gender            TEXT    NOT NULL DEFAULT '',
    nationality       TEXT    NOT NULL DEFAULT '',
    national_id       TEXT    NOT NULL DEFAULT '',
    address_line1     TEXT    NOT NULL DEFAULT '',
    address_line2     TEXT    NOT NULL DEFAULT '',
    city              TEXT    NOT NULL DEFAULT '',
    postal_code       TEXT    NOT NULL DEFAULT '',
    country_code      TEXT    NOT NULL DEFAULT '',
    photo_document_id TEXT    NULL REFERENCES documents (id) ON DELETE SET NULL,
    is_active         INTEGER NOT NULL DEFAULT 1 CHECK (is_active IN (0, 1)),
    version           INTEGER NOT NULL DEFAULT 1,
    created_at        TEXT    NOT NULL,
    updated_at        TEXT    NOT NULL
);

CREATE INDEX idx_people_organisation ON people (organisation_id);
CREATE INDEX idx_people_names ON people (organisation_id, last_name, first_name);

-- A role is scoped to one academic year: a child is a student in some years and
-- an alumnus later, and a teacher may also be a guardian in the same year.
CREATE TABLE person_roles (
    id                TEXT    PRIMARY KEY,
    organisation_id   TEXT    NOT NULL REFERENCES organisations (id) ON DELETE CASCADE,
    person_id         TEXT    NOT NULL REFERENCES people (id) ON DELETE CASCADE,
    academic_year_id  TEXT    NOT NULL REFERENCES academic_years (id) ON DELETE RESTRICT,
    role              TEXT    NOT NULL CHECK (role IN ('STUDENT', 'EMPLOYEE', 'GUARDIAN', 'SUPPLIER_CONTACT')),
    starts_on         TEXT    NULL,
    ends_on           TEXT    NULL,
    is_current        INTEGER NOT NULL DEFAULT 1 CHECK (is_current IN (0, 1)),
    version           INTEGER NOT NULL DEFAULT 1,
    created_at        TEXT    NOT NULL,
    updated_at        TEXT    NOT NULL,
    CHECK (ends_on IS NULL OR starts_on IS NULL OR ends_on >= starts_on)
);

-- One person holds a given role at most once in a given year.
CREATE UNIQUE INDEX idx_person_roles_unique ON person_roles (person_id, role, academic_year_id);
CREATE INDEX idx_person_roles_scope ON person_roles (organisation_id, academic_year_id, role);
CREATE INDEX idx_person_roles_person ON person_roles (person_id);

-- Role-specific facts. The profile row is keyed by the role, so a person has
-- exactly one student record per year they are a student in, and the columns
-- that only make sense for one role never appear on the others.
CREATE TABLE student_profiles (
    person_role_id    TEXT    PRIMARY KEY REFERENCES person_roles (id) ON DELETE CASCADE,
    student_number    TEXT    NOT NULL DEFAULT '',
    admission_date    TEXT    NULL,
    status            TEXT    NOT NULL DEFAULT 'ENROLLED' CHECK (status IN ('APPLICANT', 'ENROLLED', 'SUSPENDED', 'WITHDRAWN', 'GRADUATED')),
    previous_school   TEXT    NOT NULL DEFAULT '',
    is_boarding       INTEGER NOT NULL DEFAULT 0 CHECK (is_boarding IN (0, 1)),
    created_at        TEXT    NOT NULL,
    updated_at        TEXT    NOT NULL
);

CREATE UNIQUE INDEX idx_student_profiles_number ON student_profiles (student_number) WHERE student_number <> '';

CREATE TABLE employee_profiles (
    person_role_id    TEXT    PRIMARY KEY REFERENCES person_roles (id) ON DELETE CASCADE,
    employee_number   TEXT    NOT NULL DEFAULT '',
    job_title         TEXT    NOT NULL DEFAULT '',
    department        TEXT    NOT NULL DEFAULT '',
    hired_on          TEXT    NULL,
    ended_on          TEXT    NULL,
    payroll_group     TEXT    NOT NULL DEFAULT '',
    contract_type     TEXT    NOT NULL DEFAULT 'PERMANENT' CHECK (contract_type IN ('PERMANENT', 'FIXED_TERM', 'PART_TIME', 'CONTRACT', 'INTERN')),
    created_at        TEXT    NOT NULL,
    updated_at        TEXT    NOT NULL
);

CREATE UNIQUE INDEX idx_employee_profiles_number ON employee_profiles (employee_number) WHERE employee_number <> '';

CREATE TABLE guardian_profiles (
    person_role_id         TEXT    PRIMARY KEY REFERENCES person_roles (id) ON DELETE CASCADE,
    is_emergency_contact   INTEGER NOT NULL DEFAULT 1 CHECK (is_emergency_contact IN (0, 1)),
    may_collect_student    INTEGER NOT NULL DEFAULT 1 CHECK (may_collect_student IN (0, 1)),
    is_billing_contact     INTEGER NOT NULL DEFAULT 0 CHECK (is_billing_contact IN (0, 1)),
    can_authorise_medical  INTEGER NOT NULL DEFAULT 0 CHECK (can_authorise_medical IN (0, 1)),
    created_at             TEXT    NOT NULL,
    updated_at             TEXT    NOT NULL
);

-- One directed, typed edge covers every human relationship in the school, so no
-- domain owns a family tree: an employee who is also a guardian is simply a
-- GUARDIAN_OF edge, and a guardian who is the emergency contact for a colleague
-- is an EMERGENCY_CONTACT_OF edge pointing the other way
-- (docs/adr/ADR-011).
CREATE TABLE person_relationships (
    id                TEXT    PRIMARY KEY,
    organisation_id   TEXT    NOT NULL REFERENCES organisations (id) ON DELETE CASCADE,
    academic_year_id  TEXT    NOT NULL REFERENCES academic_years (id) ON DELETE RESTRICT,
    from_person_id    TEXT    NOT NULL REFERENCES people (id) ON DELETE CASCADE,
    to_person_id      TEXT    NOT NULL REFERENCES people (id) ON DELETE CASCADE,
    relationship_type TEXT    NOT NULL CHECK (relationship_type IN ('GUARDIAN_OF', 'EMERGENCY_CONTACT_OF', 'SPONSOR_OF', 'CAREGIVER_OF', 'NEXT_OF_KIN_OF', 'SIBLING_OF', 'SPOUSE_OF', 'STEP_PARENT_OF')),
    relationship_role TEXT    NOT NULL DEFAULT '',
    is_primary        INTEGER NOT NULL DEFAULT 0 CHECK (is_primary IN (0, 1)),
    is_active         INTEGER NOT NULL DEFAULT 1 CHECK (is_active IN (0, 1)),
    version           INTEGER NOT NULL DEFAULT 1,
    created_at        TEXT    NOT NULL,
    updated_at        TEXT    NOT NULL,
    CHECK (from_person_id <> to_person_id)
);

CREATE UNIQUE INDEX idx_relationships_unique ON person_relationships (academic_year_id, from_person_id, to_person_id, relationship_type, relationship_role);
CREATE INDEX idx_relationships_from ON person_relationships (organisation_id, academic_year_id, from_person_id, relationship_type);
CREATE INDEX idx_relationships_to ON person_relationships (organisation_id, academic_year_id, to_person_id, relationship_type);

-- At most one billing-relevant guardian per student per year, mirroring the
-- single-primary-campus rule. Finance reads this flag; it does not own it.
CREATE UNIQUE INDEX idx_relationships_single_primary_guardian ON person_relationships (academic_year_id, to_person_id) WHERE relationship_type = 'GUARDIAN_OF' AND is_primary = 1;

-- A household is a billing party, not a family tree: a couple paying for three
-- children, or a family re-grouped between years. Membership is derived from
-- GUARDIAN_OF edges at read time; this table holds only the grouping.
CREATE TABLE households (
    id                   TEXT    PRIMARY KEY,
    organisation_id      TEXT    NOT NULL REFERENCES organisations (id) ON DELETE CASCADE,
    academic_year_id     TEXT    NOT NULL REFERENCES academic_years (id) ON DELETE RESTRICT,
    name                 TEXT    NOT NULL,
    billing_email        TEXT    NOT NULL DEFAULT '',
    billing_phone        TEXT    NOT NULL DEFAULT '',
    billing_address_line1 TEXT   NOT NULL DEFAULT '',
    billing_address_line2 TEXT   NOT NULL DEFAULT '',
    billing_city         TEXT    NOT NULL DEFAULT '',
    billing_postal_code  TEXT    NOT NULL DEFAULT '',
    billing_country_code TEXT    NOT NULL DEFAULT '',
    version              INTEGER NOT NULL DEFAULT 1,
    created_at           TEXT    NOT NULL,
    updated_at           TEXT    NOT NULL,
    UNIQUE (organisation_id, academic_year_id, name)
);

CREATE INDEX idx_households_scope ON households (organisation_id, academic_year_id);

CREATE TABLE household_members (
    household_id     TEXT    NOT NULL REFERENCES households (id) ON DELETE CASCADE,
    person_id        TEXT    NOT NULL REFERENCES people (id) ON DELETE CASCADE,
    member_role      TEXT    NOT NULL DEFAULT 'OTHER' CHECK (member_role IN ('HEAD', 'PARTNER', 'DEPENDENT', 'OTHER')),
    is_billing_party INTEGER NOT NULL DEFAULT 0 CHECK (is_billing_party IN (0, 1)),
    created_at       TEXT    NOT NULL,
    PRIMARY KEY (household_id, person_id)
);

CREATE INDEX idx_household_members_person ON household_members (person_id);

CREATE TABLE user_accounts (
    id                     TEXT    PRIMARY KEY,
    organisation_id        TEXT    NOT NULL REFERENCES organisations (id) ON DELETE CASCADE,
    person_id              TEXT    REFERENCES people (id) ON DELETE SET NULL,
    username               TEXT    NOT NULL,
    email                  TEXT    NOT NULL,
    display_name           TEXT    NOT NULL DEFAULT '',
    password_hash          TEXT    NOT NULL,
    is_active              INTEGER NOT NULL DEFAULT 1 CHECK (is_active IN (0, 1)),
    is_system              INTEGER NOT NULL DEFAULT 0 CHECK (is_system IN (0, 1)),
    must_change_password   INTEGER NOT NULL DEFAULT 0 CHECK (must_change_password IN (0, 1)),
    failed_attempts        INTEGER NOT NULL DEFAULT 0,
    locked_until           TEXT    NULL,
    last_login_at          TEXT    NULL,
    version                INTEGER NOT NULL DEFAULT 1,
    created_at             TEXT    NOT NULL,
    updated_at             TEXT    NOT NULL,
    UNIQUE (organisation_id, username)
);

CREATE INDEX idx_user_accounts_org_email ON user_accounts (organisation_id, email);
CREATE INDEX idx_user_accounts_person ON user_accounts (person_id);

CREATE TABLE permissions (
    code         TEXT PRIMARY KEY,
    category     TEXT NOT NULL,
    description  TEXT NOT NULL DEFAULT ''
);

CREATE TABLE roles (
    id                TEXT    PRIMARY KEY,
    organisation_id   TEXT    NOT NULL REFERENCES organisations (id) ON DELETE CASCADE,
    code              TEXT    NOT NULL,
    name              TEXT    NOT NULL,
    description       TEXT    NOT NULL DEFAULT '',
    is_system         INTEGER NOT NULL DEFAULT 0 CHECK (is_system IN (0, 1)),
    created_at        TEXT    NOT NULL,
    updated_at        TEXT    NOT NULL,
    UNIQUE (organisation_id, code)
);

CREATE INDEX idx_roles_organisation ON roles (organisation_id);

CREATE TABLE role_permissions (
    role_id          TEXT NOT NULL REFERENCES roles (id) ON DELETE CASCADE,
    permission_code  TEXT NOT NULL REFERENCES permissions (code) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_code)
);

CREATE INDEX idx_role_permissions_permission ON role_permissions (permission_code);

CREATE TABLE user_roles (
    user_account_id  TEXT NOT NULL REFERENCES user_accounts (id) ON DELETE CASCADE,
    role_id          TEXT NOT NULL REFERENCES roles (id) ON DELETE CASCADE,
    organisation_id  TEXT NOT NULL REFERENCES organisations (id) ON DELETE CASCADE,
    granted_at       TEXT NOT NULL,
    granted_by       TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (user_account_id, role_id)
);

CREATE INDEX idx_user_roles_role ON user_roles (role_id);
CREATE INDEX idx_user_roles_organisation ON user_roles (organisation_id);

-- Append-only. There is no updated_at and no delete path: financial audit
-- history is never rewritten (docs/adr/ADR-006).
CREATE TABLE audit_entries (
    id                TEXT    PRIMARY KEY,
    organisation_id   TEXT    NOT NULL,
    campus_id         TEXT    NOT NULL DEFAULT '',
    actor_user_id     TEXT    NOT NULL DEFAULT '',
    actor_label       TEXT    NOT NULL DEFAULT '',
    entity_type       TEXT    NOT NULL,
    entity_id         TEXT    NOT NULL,
    operation         TEXT    NOT NULL,
    occurred_at       TEXT    NOT NULL,
    before_json       TEXT    NOT NULL DEFAULT '',
    after_json        TEXT    NOT NULL DEFAULT '',
    reason            TEXT    NOT NULL DEFAULT '',
    correlation_id    TEXT    NOT NULL DEFAULT '',
    site_id           TEXT    NOT NULL DEFAULT '',
    ip_address        TEXT    NOT NULL DEFAULT '',
    device            TEXT    NOT NULL DEFAULT '',
    session_id        TEXT    NOT NULL DEFAULT ''
);

CREATE INDEX idx_audit_organisation_time ON audit_entries (organisation_id, occurred_at);
CREATE INDEX idx_audit_entity ON audit_entries (entity_type, entity_id);
CREATE INDEX idx_audit_actor ON audit_entries (actor_user_id);
CREATE INDEX idx_audit_correlation ON audit_entries (correlation_id);

-- Durable outbox. Written in the same transaction as the business change and
-- published afterwards (docs/adr/ADR-007).
CREATE TABLE outbox_events (
    id                TEXT    PRIMARY KEY,
    organisation_id   TEXT    NOT NULL,
    site_id           TEXT    NOT NULL DEFAULT '',
    event_type        TEXT    NOT NULL,
    aggregate_type    TEXT    NOT NULL,
    aggregate_id      TEXT    NOT NULL,
    payload           TEXT    NOT NULL,
    correlation_id    TEXT    NOT NULL DEFAULT '',
    created_at        TEXT    NOT NULL,
    processed_at      TEXT    NOT NULL DEFAULT '',
    attempt_count     INTEGER NOT NULL DEFAULT 0,
    last_error        TEXT    NOT NULL DEFAULT ''
);

CREATE INDEX idx_outbox_pending ON outbox_events (processed_at, created_at);
CREATE INDEX idx_outbox_aggregate ON outbox_events (aggregate_type, aggregate_id);

CREATE TABLE inbox_messages (
    id                TEXT    PRIMARY KEY,
    organisation_id   TEXT    NOT NULL,
    message_id        TEXT    NOT NULL,
    payload           TEXT    NOT NULL,
    received_at       TEXT    NOT NULL,
    processed_at      TEXT    NOT NULL DEFAULT '',
    UNIQUE (message_id)
);

CREATE INDEX idx_inbox_pending ON inbox_messages (processed_at, received_at);

-- Configuration hierarchy: system -> organisation -> campus -> module -> user.
CREATE TABLE settings (
    scope_type    TEXT NOT NULL CHECK (scope_type IN ('SYSTEM', 'ORGANISATION', 'CAMPUS', 'MODULE', 'USER')),
    scope_id      TEXT NOT NULL DEFAULT '',
    key           TEXT NOT NULL,
    value         TEXT NOT NULL,
    value_type    TEXT NOT NULL DEFAULT 'string' CHECK (value_type IN ('string', 'integer', 'boolean', 'decimal', 'json')),
    organisation_id TEXT NOT NULL DEFAULT '',
    version       INTEGER NOT NULL DEFAULT 1,
    updated_at    TEXT NOT NULL,
    updated_by    TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (scope_type, scope_id, key)
);

CREATE INDEX idx_settings_organisation ON settings (organisation_id);
