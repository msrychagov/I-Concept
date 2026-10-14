-- Runtime records are separate rows. JSONB preserves versioned test snapshots
-- and arbitrary form answers; it is not a copy of the static question banks.
CREATE TABLE vucs
(
    id       text PRIMARY KEY,
    position bigint                                               NOT NULL,
    payload  jsonb                                                NOT NULL CHECK (jsonb_typeof(payload) = 'object'),
    name     text GENERATED ALWAYS AS (payload ->> 'name') STORED NOT NULL,
    CHECK (id = payload ->> 'id' AND length(id) > 0)
);
CREATE UNIQUE INDEX vucs_name_unique ON vucs (lower(name));

CREATE TABLE platoons
(
    id       text PRIMARY KEY,
    position bigint                                               NOT NULL,
    payload  jsonb                                                NOT NULL,
    name     text GENERATED ALWAYS AS (payload ->> 'name') STORED NOT NULL,
    vuc      text GENERATED ALWAYS AS (payload ->> 'vuc') STORED  NOT NULL,
    CHECK (id = payload ->> 'id' AND length(id) > 0)
);
CREATE UNIQUE INDEX platoons_vuc_name_unique ON platoons (lower(vuc), lower(name));

CREATE TABLE accounts
(
    id         text PRIMARY KEY,
    position   bigint                                                NOT NULL,
    payload    jsonb                                                 NOT NULL,
    role       text GENERATED ALWAYS AS (payload ->> 'role') STORED  NOT NULL CHECK (role IN ('teacher', 'student')),
    email      text GENERATED ALWAYS AS (payload ->> 'email') STORED NOT NULL,
    owner_id   text GENERATED ALWAYS AS (NULLIF(payload ->> 'ownerId', '')) STORED,
    platoon_id text GENERATED ALWAYS AS (NULLIF(payload ->> 'platoonId', '')) STORED,
    vuc        text GENERATED ALWAYS AS (payload ->> 'vuc') STORED,
    CHECK (id = payload ->> 'id' AND length(id) > 0),
    FOREIGN KEY (platoon_id) REFERENCES platoons (id) DEFERRABLE INITIALLY DEFERRED
);
CREATE UNIQUE INDEX accounts_role_email_unique ON accounts (role, lower(email));
CREATE INDEX accounts_owner_idx ON accounts (owner_id);
CREATE INDEX accounts_vuc_idx ON accounts (lower(vuc));
CREATE INDEX accounts_platoon_idx ON accounts (platoon_id);

CREATE TABLE profiles
(
    id       text PRIMARY KEY,
    position bigint NOT NULL,
    payload  jsonb  NOT NULL,
    owner_id text GENERATED ALWAYS AS (NULLIF(payload ->> 'ownerId', '')) STORED,
    CHECK (id = payload ->> 'id' AND length(id) > 0)
);
CREATE INDEX profiles_owner_idx ON profiles (owner_id);

CREATE TABLE assignments
(
    id       text PRIMARY KEY,
    position bigint NOT NULL,
    payload  jsonb  NOT NULL,
    owner_id text GENERATED ALWAYS AS (NULLIF(payload ->> 'ownerId', '')) STORED,
    form_id  text GENERATED ALWAYS AS (NULLIF(payload ->> 'formId', '')) STORED,
    CHECK (id = payload ->> 'id' AND length(id) > 0)
);
CREATE INDEX assignments_owner_idx ON assignments (owner_id);

CREATE TABLE logins
(
    id         text PRIMARY KEY,
    position   bigint                                                    NOT NULL,
    payload    jsonb                                                     NOT NULL,
    account_id text GENERATED ALWAYS AS (payload ->> 'studentId') STORED NOT NULL,
    CHECK (id = payload ->> 'hash' AND length(id) > 0),
    FOREIGN KEY (account_id) REFERENCES accounts (id) DEFERRABLE INITIALLY DEFERRED
);
CREATE INDEX logins_account_idx ON logins (account_id);

CREATE TABLE invitations
(
    id         text PRIMARY KEY,
    position   bigint                                               NOT NULL,
    payload    jsonb                                                NOT NULL,
    role       text GENERATED ALWAYS AS (payload ->> 'role') STORED NOT NULL CHECK (role IN ('teacher', 'student')),
    issuer_id  text GENERATED ALWAYS AS (NULLIF(payload ->> 'issuerId', '')) STORED,
    platoon_id text GENERATED ALWAYS AS (NULLIF(payload ->> 'platoonId', '')) STORED,
    CHECK (id = payload ->> 'hash' AND length(id) > 0),
    FOREIGN KEY (issuer_id) REFERENCES accounts (id) DEFERRABLE INITIALLY DEFERRED,
    FOREIGN KEY (platoon_id) REFERENCES platoons (id) DEFERRABLE INITIALLY DEFERRED
);
CREATE INDEX invitations_issuer_idx ON invitations (issuer_id);
CREATE INDEX invitations_platoon_idx ON invitations (platoon_id);

CREATE TABLE sessions
(
    id            text PRIMARY KEY,
    position      bigint NOT NULL,
    payload       jsonb  NOT NULL,
    student_id    text GENERATED ALWAYS AS (NULLIF(payload -> 'session' ->> 'studentId', '')) STORED,
    assignment_id text GENERATED ALWAYS AS (NULLIF(payload -> 'session' ->> 'assignmentId', '')) STORED,
    teacher_id    text GENERATED ALWAYS AS (NULLIF(payload -> 'session' ->> 'teacherId', '')) STORED,
    CHECK (id = payload -> 'session' ->> 'id' AND length(id) > 0)
);
-- Imported diagnostic archives retain IDs from their source installation.
-- Those accounts/assignments need not exist here; access uses teacher_id.
CREATE INDEX sessions_student_assignment_idx ON sessions (student_id, assignment_id);
CREATE INDEX sessions_teacher_idx ON sessions (teacher_id);

CREATE TABLE form_submissions
(
    id            text PRIMARY KEY,
    position      bigint                                                       NOT NULL,
    payload       jsonb                                                        NOT NULL,
    student_id    text GENERATED ALWAYS AS (payload ->> 'studentId') STORED    NOT NULL,
    assignment_id text GENERATED ALWAYS AS (payload ->> 'assignmentId') STORED NOT NULL,
    CHECK (id = payload ->> 'id' AND length(id) > 0),
    FOREIGN KEY (student_id) REFERENCES accounts (id) DEFERRABLE INITIALLY DEFERRED,
    FOREIGN KEY (assignment_id) REFERENCES assignments (id) DEFERRABLE INITIALLY DEFERRED,
    UNIQUE (student_id, assignment_id)
);

CREATE TABLE repository_backups
(
    id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    created_at timestamptz NOT NULL DEFAULT now(),
    payload    jsonb       NOT NULL
);
