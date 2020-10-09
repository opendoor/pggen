DROP SCHEMA public CASCADE;
CREATE SCHEMA public;

CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email text NOT NULL,
    -- pggen can generate shims for json values
    bio json NOT NULL,
    -- ... and for jsonb values
    config jsonb NOT NULL,
    homepage jsonb NOT NULL
);
