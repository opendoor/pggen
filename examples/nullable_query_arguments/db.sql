DROP SCHEMA public CASCADE;
CREATE SCHEMA public;

CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email text NOT NULL,
    nickname text
);
