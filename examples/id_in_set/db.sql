DROP SCHEMA public CASCADE;
CREATE SCHEMA public;

CREATE TABLE foos (
    id SERIAL PRIMARY KEY,
    value text
);
