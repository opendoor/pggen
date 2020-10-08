DROP SCHEMA public CASCADE;
CREATE SCHEMA public;

CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    -- The UNIQUE constraint is importaint here because postgres will use it
    -- to detect when there is a conflict on insert.
    email text NOT NULL UNIQUE,
    slogan text NOT NULL,
    rating text NOT NULL
);
