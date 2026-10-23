-- Instructors of a center as students name them in texts. An entry is not an
-- account: some instructors never register.
CREATE TABLE instructors
(
    id       text PRIMARY KEY,
    position bigint                                              NOT NULL,
    payload  jsonb                                               NOT NULL,
    vuc      text GENERATED ALWAYS AS (payload ->> 'vuc') STORED NOT NULL,
    CHECK (id = payload ->> 'id' AND length(id) > 0)
);
CREATE INDEX instructors_vuc_idx ON instructors (lower(vuc));
