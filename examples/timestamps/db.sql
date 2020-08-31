DROP SCHEMA public CASCADE;
CREATE SCHEMA public;

CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email text NOT NULL,
    created_at timestamp NOT NULL,
    updated_at timestamp NOT NULL,
    deleted_at timestamp
);
