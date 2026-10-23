-- Cohorts (graduating classes): a named set of platoons of one center that
-- receives an essay together.
CREATE TABLE cohorts
(
    id       text PRIMARY KEY,
    position bigint                                              NOT NULL,
    payload  jsonb                                               NOT NULL,
    vuc      text GENERATED ALWAYS AS (payload ->> 'vuc') STORED NOT NULL,
    CHECK (id = payload ->> 'id' AND length(id) > 0)
);
CREATE INDEX cohorts_vuc_idx ON cohorts (lower(vuc));
