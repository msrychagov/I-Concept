-- Texts written by a language model. A record belongs to the content of the
-- input, not to a student, so it has no foreign keys.
CREATE TABLE portraits
(
    id       text PRIMARY KEY,
    position bigint NOT NULL,
    payload  jsonb  NOT NULL,
    task     text GENERATED ALWAYS AS (payload ->> 'task') STORED NOT NULL,
    CHECK (id = payload ->> 'id' AND length(id) > 0)
);
CREATE INDEX portraits_task_idx ON portraits (task);
