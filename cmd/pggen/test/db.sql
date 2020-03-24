DROP SCHEMA public CASCADE;
CREATE SCHEMA public;

-- enable uuid utils
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

--
-- Create the tables
--

CREATE TYPE enum_type as ENUM ('option1', 'option2');
CREATE TABLE type_rainbow (
    id SERIAL PRIMARY KEY NOT NULL,

    -- string types
    text_field TEXT,
    text_field_not_null TEXT NOT NULL,
    varchar_field VARCHAR(40),
    varchar_field_not_null VARCHAR(40) NOT NULL,
    char_field CHAR(40),
    char_field_not_null CHAR(40) NOT NULL,

    -- money
    money_field MONEY,
    money_field_not_null MONEY NOT NULL,

    -- date and time types
    ts_field TIMESTAMP,
    ts_field_not_null TIMESTAMP NOT NULL,
    tsz_field TIMESTAMP WITH TIME ZONE,
    tsz_field_not_null TIMESTAMP WITH TIME ZONE NOT NULL,
    date_field DATE,
    date_field_not_null DATE NOT NULL,
    time_field TIME,
    time_field_not_null TIME NOT NULL,
    timez_field TIME WITH TIME ZONE,
    timez_field_not_null TIME WITH TIME ZONE NOT NULL,
    interval_field INTERVAL,
    interval_field_not_null INTERVAL NOT NULL,

    -- boolean type
    bool_field BOOLEAN,
    bool_field_not_null BOOLEAN NOT NULL,

    -- enum type
    enum_field enum_type,
    enum_field_not_null enum_type NOT NULL,

    -- uuid type
    uuid_field UUID,
    uuid_field_not_null UUID NOT NULL,

    -- binary blob types
    bytea_field BYTEA,
    bytea_field_not_null BYTEA NOT NULL,

    -- numeric types
    smallint_field SMALLINT,
    smallint_field_not_null SMALLINT NOT NULL,
    integer_field INTEGER,
    integer_field_not_null INTEGER NOT NULL,
    bigint_field BIGINT,
    bigint_field_not_null BIGINT NOT NULL,
    decimal_field DECIMAL,
    decimal_field_not_null DECIMAL NOT NULL,
    numeric_field NUMERIC,
    numeric_field_not_null NUMERIC NOT NULL,
    numeric_prec_field NUMERIC(10),
    numeric_prec_field_not_null NUMERIC(10) NOT NULL,
    numeric_prec_scale_field NUMERIC(10, 2),
    numeric_prec_scale_field_not_null NUMERIC(10, 2) NOT NULL,
    real_field REAL,
    real_field_not_null REAL NOT NULL,
    double_field DOUBLE PRECISION,
    double_field_not_null DOUBLE PRECISION NOT NULL,
    serial_field SERIAL,
    serial_field_not_null SERIAL NOT NULL,
    big_serial_field BIGSERIAL,
    big_serial_field_not_null BIGSERIAL NOT NULL


    -- TODO: postgis types
    -- TODO: geometric types
    -- TODO: network address types
    -- TODO: bit string types
    -- TODO: text search types
    -- TODO: XML type
    -- TODO: JSON type
    -- TODO: arrays
    -- TODO: composite types
    -- TODO: range types
    -- TODO: object identifier types
);

CREATE TABLE small_entities (
	id SERIAL PRIMARY KEY NOT NULL,
    anint integer NOT NULL
);

CREATE TABLE attachments (
    id UUID PRIMARY KEY NOT NULL DEFAULT uuid_generate_v4(),
    small_entity_id integer NOT NULL
        REFERENCES small_entities(id) ON DELETE RESTRICT ON UPDATE CASCADE,
    value text
);

CREATE TABLE single_attachments (
    id SERIAL PRIMARY KEY NOT NULL,
    -- pggen will be able to infer that `small_entities` has 0 or 1
    -- `single_attachments` because of the UNIQUE index on the foreign key
    small_entity_id integer NOT NULL UNIQUE
        REFERENCES small_entities(id) ON DELETE RESTRICT ON UPDATE CASCADE,
    created_at timestamp with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE no_infer (
    id SERIAL PRIMARY KEY NOT NULL,
    small_entity_id integer NOT NULL
        REFERENCES small_entities(id) ON DELETE RESTRICT ON UPDATE CASCADE
);

CREATE TABLE explicit_belongs_to (
    id SERIAL PRIMARY KEY NOT NULL,
    small_entity_id integer NOT NULL
);

CREATE TABLE explicit_belongs_to_many (
    id SERIAL PRIMARY KEY NOT NULL,
    small_entity_id integer NOT NULL
);

CREATE TABLE array_members (
    id SERIAL PRIMARY KEY,
    text_array text[] NOT NULL,
    int_array int[]
);

CREATE TABLE timestamps_both (
    id SERIAL PRIMARY KEY,
    created_at timestamp,
    updated_at timestamp NOT NULL,
    payload text
);

CREATE TABLE timestamps_global (
    id SERIAL PRIMARY KEY,
    created_at timestamp,
    updated_at timestamp NOT NULL,
    payload text
);

CREATE TABLE timestamps_just_created (
    id SERIAL PRIMARY KEY,
    made_at timestamp NOT NULL,
    payload text
);

CREATE TABLE timestamps_just_updated (
    id SERIAL PRIMARY KEY,
    last_touched timestamp with time zone,
    payload text
);

-- pggen ought not rely on the column ordering of a table, as migrations
-- might not have been applied exactly in order.
CREATE TABLE col_order (
    id SERIAL PRIMARY KEY,
    field1 text NOT NULL,
    field2 int NOT NULL,
    field3 int NOT NULL
);

-- SQL has the best identifier rules. Sigh. Let's support them.
-- At least you don't seem to be able to start with a number
-- without quoting.
CREATE TABLE "Weird NaMeS" (
    "even id is weird" SERIAL PRIMARY KEY,
    WeAreTalking___REALLY_badstyle integer NOT NULL,
    "Got Whitespace?" text NOT NULL,
    "But
    Why
    Tho" integer
);

CREATE TABLE "Weird?! Kid" (
    "space id" SERIAL PRIMARY KEY,
    "Daddy" integer NOT NULL
        REFERENCES "Weird NaMeS"("even id is weird")
            ON DELETE RESTRICT ON UPDATE CASCADE
);

--
-- Load Data
--

INSERT INTO type_rainbow (
    -- string types
    text_field, text_field_not_null, varchar_field, varchar_field_not_null, char_field, char_field_not_null,

    -- money
    money_field, money_field_not_null,

    -- date and time types
    ts_field, ts_field_not_null, tsz_field, tsz_field_not_null, date_field, date_field_not_null,
    time_field, time_field_not_null, timez_field, timez_field_not_null, interval_field,
    interval_field_not_null,

    -- boolean type
    bool_field, bool_field_not_null,

    -- enum type
    enum_field, enum_field_not_null,

    -- uuid type
    uuid_field, uuid_field_not_null,

    bytea_field, bytea_field_not_null,

    -- numeric types
    smallint_field, smallint_field_not_null, integer_field, integer_field_not_null, bigint_field,
    bigint_field_not_null, decimal_field, decimal_field_not_null, numeric_field,
    numeric_field_not_null, numeric_prec_field, numeric_prec_field_not_null, numeric_prec_scale_field,
    numeric_prec_scale_field_not_null, real_field, real_field_not_null, double_field,
    double_field_not_null
    -- serial fields are left to auto-increment
) VALUES (
    'foo', 'foo', 'foo', 'foo', 'foo', 'foo',

    -- money
    '3.50'::float::numeric::money, '3.50'::float::numeric::money,

    -- date and time types
    TIMESTAMP '1999-01-08 04:05:06',
    TIMESTAMP '1999-01-08 04:05:06',
    TIMESTAMP WITH TIME ZONE '1999-01-08 04:05:06+03',
    TIMESTAMP WITH TIME ZONE '1999-01-08 04:05:06+03',
    DATE '1995-05-19',
    DATE '1995-05-19',
    TIME '03:11:21',
    TIME '03:11:21',
    TIME WITH TIME ZONE '08:00:00+03',
    TIME WITH TIME ZONE '08:00:00+03',
    INTERVAL '1 02:00:00',
    INTERVAL '1 02:00:00',

    -- boolean type
    'true'::boolean, 'false'::boolean,

    -- enum type
    'option1', 'option2',

    -- uuid type
    'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11',

    -- binary blob types
    '\xdeadbeef', '\xdeadbeef',

    -- numeric types
    1, 1, 2, 2, 3, 3, 15.4, 15.4, 16.4, 16.4, 999, 999, 19.99, 19.99, 2.3, 2.3, 9.3, 9.3
);

INSERT INTO type_rainbow (
    -- string types
    text_field_not_null, varchar_field_not_null, char_field_not_null,

    -- money
    money_field_not_null,

    -- date and time types
    ts_field_not_null, tsz_field_not_null, date_field_not_null,
    time_field_not_null, timez_field_not_null, interval_field_not_null,

    -- boolean type
    bool_field_not_null,

    -- enum type
    enum_field_not_null,

    -- uuid type
    uuid_field_not_null,

    -- binary blob type
    bytea_field_not_null,

    -- numeric types
    smallint_field_not_null, integer_field_not_null,
    bigint_field_not_null, decimal_field_not_null,
    numeric_field_not_null, numeric_prec_field_not_null,
    numeric_prec_scale_field_not_null, real_field_not_null,
    double_field_not_null
    -- serial fields are left to auto-increment
) VALUES (
    'foo', 'foo', 'foo',

    -- money
    '3.50'::float::numeric::money,

    -- date and time types
    TIMESTAMP '1999-01-08 04:05:06',
    TIMESTAMP WITH TIME ZONE '1999-01-08 04:05:06+03',
    DATE '1995-05-19',
    TIME '03:11:21',
    TIME WITH TIME ZONE '08:00:00+03',
    INTERVAL '1 02:00:00',

    -- boolean type
    'true'::boolean,

    -- enum type
    'option1',

    -- uuid type
    'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11',

    -- binary blob type
    '\xdeadbeef',

    -- numeric types
    1, 2, 3, 15.4, 16.4, 999, 19.99, 2.3, 9.3
);

INSERT INTO small_entities (anint) VALUES (17);

INSERT INTO attachments(small_entity_id, value)
    SELECT id, 'text 1' FROM small_entities;

INSERT INTO attachments(small_entity_id, value)
    SELECT id, 'text 2' FROM small_entities;

INSERT INTO attachments(small_entity_id, value)
    SELECT id, 'text 3' FROM small_entities;

--
-- Create the stored functions
--

CREATE OR REPLACE FUNCTION returns_text()
RETURNS TEXT
AS $$
BEGIN
    RETURN 'foo';
END
$$
LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION concats_text(t1 TEXT, t2 TEXT)
RETURNS TEXT
AS $$
BEGIN
    RETURN t1 || t2;
END
$$
LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION select_string_types()
RETURNS TABLE (
    text_field TEXT,
    text_field_not_null TEXT,
    varchar_field VARCHAR(40),
    varchar_field_not_null VARCHAR(40),
    char_field CHAR(40),
    char_field_not_null CHAR(40)
) AS $$
BEGIN
    RETURN QUERY
    SELECT t.text_field, t.text_field_not_null,
           t.varchar_field, t.varchar_field_not_null,
           t.char_field, t.char_field_not_null
    FROM type_rainbow t;
END
$$
LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION select_matching_string(
    must_be TEXT
)
RETURNS TABLE (
    text_field TEXT
)
AS $$
SELECT t.text_field
FROM type_rainbow t
WHERE t.text_field = must_be;
$$
LANGUAGE sql;

CREATE OR REPLACE FUNCTION select_money()
RETURNS TABLE (
    money_field MONEY,
    money_field_not_null MONEY
)
AS $$
SELECT money_field, money_field_not_null from type_rainbow;
$$
LANGUAGE sql;

CREATE OR REPLACE FUNCTION select_time()
RETURNS TABLE (
    ts_field TIMESTAMP,
    ts_field_not_null TIMESTAMP,
    tsz_field TIMESTAMP WITH TIME ZONE,
    tsz_field_not_null TIMESTAMP WITH TIME ZONE,
    date_field DATE,
    date_field_not_null DATE,
    time_field TIME,
    time_field_not_null TIME,
    timez_field TIME WITH TIME ZONE,
    timez_field_not_null TIME WITH TIME ZONE
    -- TODO: intervals (lib/pq and database/sql don't have good support)
    -- interval_field INTERVAL,
    -- interval_field_not_null INTERVAL
)
AS $$
SELECT
    ts_field,
    ts_field_not_null,
    tsz_field,
    tsz_field_not_null,
    date_field,
    date_field_not_null,
    time_field,
    time_field_not_null,
    timez_field,
    timez_field_not_null
    -- interval_field,
    -- interval_field_not_null
FROM type_rainbow;
$$
LANGUAGE sql;

CREATE OR REPLACE FUNCTION select_bool()
RETURNS TABLE (
    bool_field BOOLEAN,
    bool_field_not_null BOOLEAN
) AS $$
SELECT bool_field, bool_field_not_null FROM type_rainbow;
$$
LANGUAGE sql;

CREATE OR REPLACE FUNCTION select_enum()
RETURNS TABLE (
    enum_field enum_type,
    enum_field_not_null enum_type
) AS $$
SELECT enum_field, enum_field_not_null FROM type_rainbow;
$$
LANGUAGE sql;

CREATE OR REPLACE FUNCTION select_uuid()
RETURNS TABLE (
    uuid_field UUID,
    uuid_field_not_null UUID
) AS $$
SELECT uuid_field, uuid_field_not_null FROM type_rainbow;
$$
LANGUAGE sql;

CREATE OR REPLACE FUNCTION select_blobs()
RETURNS TABLE (
    bytea_field BYTEA,
    bytea_field_not_null BYTEA
) AS $$
SELECT bytea_field, bytea_field_not_null FROM type_rainbow;
$$
LANGUAGE sql;

CREATE OR REPLACE FUNCTION select_numbers()
RETURNS TABLE (
    smallint_field SMALLINT,
    smallint_field_not_null SMALLINT,
    integer_field INTEGER,
    integer_field_not_null INTEGER,
    bigint_field BIGINT,
    bigint_field_not_null BIGINT,
    decimal_field DECIMAL,
    decimal_field_not_null DECIMAL,
    numeric_field NUMERIC,
    numeric_field_not_null NUMERIC,
    numeric_prec_field NUMERIC(10),
    numeric_prec_field_not_null NUMERIC(10),
    numeric_prec_scale_field NUMERIC(10, 2),
    numeric_prec_scale_field_not_null NUMERIC(10, 2),
    real_field REAL,
    real_field_not_null REAL,
    double_field DOUBLE PRECISION,
    double_field_not_null DOUBLE PRECISION,
    serial_field INTEGER,
    serial_field_not_null INTEGER,
    big_serial_field BIGINT,
    big_serial_field_not_null BIGINT
) AS $$
SELECT
    smallint_field,
    smallint_field_not_null,
    integer_field,
    integer_field_not_null,
    bigint_field,
    bigint_field_not_null,
    decimal_field,
    decimal_field_not_null,
    numeric_field,
    numeric_field_not_null,
    numeric_prec_field,
    numeric_prec_field_not_null,
    numeric_prec_scale_field,
    numeric_prec_scale_field_not_null,
    real_field,
    real_field_not_null,
    double_field,
    double_field_not_null,
    serial_field,
    serial_field_not_null,
    big_serial_field,
    big_serial_field_not_null
FROM type_rainbow;
$$
LANGUAGE sql;

CREATE OR REPLACE FUNCTION get_small_entity_1()
RETURNS TABLE (
	id integer,
    anint integer
) AS $$
SELECT * FROM small_entities
$$
LANGUAGE sql;

CREATE OR REPLACE FUNCTION get_small_entity_2()
RETURNS TABLE (
	id integer,
    anint integer
) AS $$
SELECT * FROM small_entities
$$
LANGUAGE sql;

-- Don't do this irl. We just need to be able to force errors within
-- the db for testing purposes.
CREATE OR REPLACE FUNCTION inject_sql(indection text)
RETURNS integer
AS $$
BEGIN
    EXECUTE(injection);
    RETURN 1;
END
$$
LANGUAGE plpgsql;
