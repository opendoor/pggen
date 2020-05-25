DROP SCHEMA public CASCADE;
CREATE SCHEMA public;

CREATE TYPE size_category AS ENUM ('small', 'large');

CREATE TABLE dogs (
    id SERIAL PRIMARY KEY,
    breed text NOT NULL,
    size size_category NOT NULL,
    age_in_dog_years integer NOT NULL
);
