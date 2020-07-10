DROP SCHEMA public CASCADE;
CREATE SCHEMA public;

CREATE TABLE grandparents (
    id SERIAL PRIMARY KEY,
    name text NOT NULL
);

CREATE TABLE parents (
    id SERIAL PRIMARY KEY,
    grandparent_id integer NOT NULL
        REFERENCES grandparents(id) ON DELETE RESTRICT ON UPDATE CASCADE,
    name text NOT NULL
);

CREATE TABLE children (
    id SERIAL PRIMARY KEY,
    parent_id integer NOT NULL
        REFERENCES parents(id) ON DELETE RESTRICT ON UPDATE CASCADE,
    name text NOT NULL
);

ALTER TABLE grandparents ADD COLUMN
    favorite_grandkid_id integer REFERENCES children(id) ON DELETE RESTRICT ON UPDATE CASCADE;
