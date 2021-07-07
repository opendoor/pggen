----------------------------------------------------------------------------------------------------
--                                                                                                --
--                                          public                                                --
--                                                                                                --
----------------------------------------------------------------------------------------------------

DROP SCHEMA IF EXISTS public CASCADE;
CREATE SCHEMA public;

-- enable uuid utils
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

--
-- Create the tables
--

CREATE TYPE enum_type AS ENUM ('', 'option1', 'option2');
CREATE TYPE enum_type_with_blank AS ENUM ('', 'blank', 'other');

CREATE TYPE funky_name_enum AS ENUM ('has spaces', '+special@chars*^?/:%()={}[]~`&''.,><', 'foo', 'foo+');

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
    big_serial_field_not_null BIGSERIAL NOT NULL,

    -- json types
    json_field json,
    json_field_not_null json NOT NULL,
    jsonb_field jsonb,
    jsonb_field_not_null jsonb NOT NULL,

    int4range_field int4range,
    int4range_field_not_null int4range NOT NULL

    -- TODO: postgis types
    -- TODO: geometric types
    -- TODO: network address types
    -- TODO: bit string types
    -- TODO: text search types
    -- TODO: XML type
    -- TODO: composite types
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

CREATE TABLE nullable_attachments (
    id SERIAL PRIMARY KEY NOT NULL,
    small_entity_id integer REFERENCES small_entities(id),
    value text NOT NULL
);

CREATE TABLE nullable_single_attachments (
    id SERIAL PRIMARY KEY NOT NULL,
    small_entity_id integer UNIQUE REFERENCES small_entities(id),
    value text NOT NULL
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
    dropped text,
    field2 int NOT NULL,
    field3 int NOT NULL
);
ALTER TABLE col_order DROP COLUMN dropped;

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

CREATE TABLE constraints (
    id SERIAL PRIMARY KEY,
    snowflake int NOT NULL UNIQUE,
    other int NOT NULL
);

CREATE TABLE text_array (
    id SERIAL PRIMARY KEY,
    value text[]
);

CREATE TABLE enum_blanks (
    id SERIAL PRIMARY KEY,
    value enum_type_with_blank NOT NULL
);

-- cyclical references
CREATE TABLE cycle1 (
    id SERIAL PRIMARY KEY,
    value text NOT NULL
);
CREATE TABLE cycle2 (
    id SERIAL PRIMARY KEY,
    value int NOT NULL,
    cycle1_id integer NOT NULL REFERENCES cycle1(id)
);
ALTER TABLE cycle1 ADD COLUMN cycle2_id integer REFERENCES cycle2(id);

-- an object tree with a cycle in the branches that is reached by multiple
-- branch paths
CREATE TABLE cycle_tree_root (
    id SERIAL PRIMARY KEY,
    value text NOT NULL
);
CREATE TABLE cycle_tree_branch1 (
    id SERIAL PRIMARY KEY,
    value text NOT NULL,
    cycle_tree_root_id integer NOT NULL REFERENCES cycle_tree_root(id)
);
CREATE TABLE cycle_tree_branch2 (
    id SERIAL PRIMARY KEY,
    value text NOT NULL,
    cycle_tree_root_id integer NOT NULL UNIQUE REFERENCES cycle_tree_root(id)
);
CREATE TABLE cycle_tree_cycle1 (
    id SERIAL PRIMARY KEY,
    value text NOT NULL,
    cycle_tree_branch1_id integer NOT NULL UNIQUE REFERENCES cycle_tree_branch1(id)
);
CREATE TABLE cycle_tree_cycle2 (
    id SERIAL PRIMARY KEY,
    value text NOT NULL,
    cycle_tree_cycle1_id integer NOT NULL UNIQUE REFERENCES cycle_tree_cycle1(id),
    cycle_tree_branch2_id integer NOT NULL UNIQUE REFERENCES cycle_tree_branch2(id)
);
CREATE TABLE cycle_tree_cycle3 (
    id SERIAL PRIMARY KEY,
    value text NOT NULL,
    cycle_tree_cycle2_id integer NOT NULL UNIQUE REFERENCES cycle_tree_cycle2(id)
);
ALTER TABLE cycle_tree_cycle1 ADD COLUMN
    cycle_tree_cycle3_id integer REFERENCES cycle_tree_cycle3(id);

CREATE TABLE simple_enums (
    id SERIAL PRIMARY KEY,
    value enum_type
);
INSERT INTO simple_enums (value) VALUES (''), ('option1'), ('option1'), ('option2'), (NULL);

-- to test that we can gracefully handle a table being modified to have an additional column
CREATE TABLE will_get_new_column (
    id SERIAL PRIMARY KEY,
    f1 text NOT NULL
);

-- to test inserting when the primary key cannot be automatically computed by the database
CREATE TABLE non_default_pkey (
    id text PRIMARY KEY,
    val integer
);

CREATE TABLE alternative_reference_name (
    id SERIAL PRIMARY KEY NOT NULL,
    small_entity_id integer NOT NULL
        REFERENCES small_entities(id) ON DELETE RESTRICT ON UPDATE CASCADE
);
CREATE TABLE alternative_reference_name_1to1 (
    id SERIAL PRIMARY KEY NOT NULL,
    small_entity_id integer NOT NULL
        REFERENCES small_entities(id) ON DELETE RESTRICT ON UPDATE CASCADE
);

CREATE TABLE custom_default_uuids (
    id SERIAL PRIMARY KEY NOT NULL,
    uuid uuid NOT NULL DEFAULT uuid_generate_v4()
);

CREATE TABLE wacky_roots (
    id SERIAL PRIMARY KEY NOT NULL,
    value text NOT NULL
);
CREATE TABLE wacky_attachments (
    id SERIAL PRIMARY KEY NOT NULL,
    value text NOT NULL,
    wacky_ref integer NOT NULL
        REFERENCES wacky_roots(id) ON DELETE RESTRICT ON UPDATE CASCADE
);
CREATE TABLE wacky_single_attachments (
    id SERIAL PRIMARY KEY NOT NULL,
    value text NOT NULL,
    wacky_ref integer NOT NULL UNIQUE
        REFERENCES wacky_roots(id) ON DELETE RESTRICT ON UPDATE CASCADE
);

CREATE TABLE gorm_defaults (
    id SERIAL PRIMARY KEY NOT NULL,
    uuid uuid NOT NULL DEFAULT uuid_generate_v4()
);

CREATE TABLE drop_cols (
    id SERIAL PRIMARY KEY NOT NULL,
    f1 int NOT NULL DEFAULT 1,
    f2 int NOT NULL
);

CREATE TABLE soft_deletables (
    id SERIAL PRIMARY KEY,
    value text NOT NULL,
    deleted_ts timestamp
);
CREATE TABLE deletable_leafs (
    id SERIAL PRIMARY KEY,
    value text NOT NULL,
    soft_deletable_id integer NOT NULL
        REFERENCES soft_deletables(id) ON DELETE RESTRICT ON UPDATE CASCADE,
    deleted_at timestamp
);

-- just a dummy table that we can run SQL on through a layer of middleware
CREATE TABLE middleware_test_recs (
    id SERIAL PRIMARY KEY,
    value text NOT NULL
);

CREATE TABLE funky_enums (
    id SERIAL PRIMARY KEY,
    enum_val funky_name_enum NOT NULL
);

-- for exclusive use by the TestOffsetTableFilling test,
-- will have columns added and removed during testing
CREATE TABLE offset_table_fillings (
    id SERIAL PRIMARY KEY,
    i1 integer NOT NULL
);

CREATE TABLE default_values (
    id SERIAL PRIMARY KEY,
    defaulted_string text NOT NULL DEFAULT 'default value',
    defaulted_int integer NOT NULL DEFAULT 42,
    nondefault_string text NOT NULL UNIQUE
);

-- for testing the `json_types` config key
CREATE TABLE json_values (
    id SERIAL PRIMARY KEY,
    json_field json,
    json_field_not_null json NOT NULL,
    jsonb_field jsonb,
    jsonb_field_not_null jsonb NOT NULL
);

CREATE TABLE double_references (
    id SERIAL PRIMARY KEY,
    sekey1 int NOT NULL REFERENCES small_entities(id) ON UPDATE CASCADE,
    sekey2 int NOT NULL REFERENCES small_entities(id) ON UPDATE CASCADE
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
    double_field_not_null,
    -- serial fields are left to auto-increment

    -- json types
    json_field, json_field_not_null, jsonb_field, jsonb_field_not_null,

    -- range types
    int4range_field, int4range_field_not_null
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
    1, 1, 2, 2, 3, 3, 15.4, 15.4, 16.4, 16.4, 999, 999, 19.99, 19.99, 2.3, 2.3, 9.3, 9.3,

    -- json types
    '5'::json, '[1, 2, "foo", null]'::json,
    '{"bar": "baz"}'::json, '{"foo": ["bar", 1]}'::json,

    -- range types
    int4range(10, 20), int4range(10, 20)
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
    double_field_not_null,
    -- serial fields are left to auto-increment

    -- json types
    json_field_not_null, jsonb_field_not_null,

    -- range types
    int4range_field_not_null
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
    1, 2, 3, 15.4, 16.4, 999, 19.99, 2.3, 9.3,

    -- json types
    '[1, 2, "foo", null]'::json, '{"foo": ["bar", 1]}'::json,

    -- range types
    int4range(10, 20)
);

INSERT INTO small_entities (anint) VALUES (17);

INSERT INTO attachments(small_entity_id, value)
    SELECT id, 'text 1' FROM small_entities;

INSERT INTO attachments(small_entity_id, value)
    SELECT id, 'text 2' FROM small_entities;

INSERT INTO attachments(small_entity_id, value)
    SELECT id, 'text 3' FROM small_entities;

--
-- Create stored functions
--

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

----------------------------------------------------------------------------------------------------
--                                                                                                --
--                                        otherschema                                             --
--                                                                                                --
----------------------------------------------------------------------------------------------------

DROP SCHEMA IF EXISTS otherschema CASCADE;
CREATE SCHEMA otherschema;

-- NOTE: name clashes with enum defined in public. This should not cause any problems
--       because they are in different schemas.
CREATE TYPE otherschema.enum_type AS ENUM ('opt1', 'opt2', 'opt3');

CREATE TABLE otherschema.foos (
    id SERIAL PRIMARY KEY,
    value text NOT NULL,
    my_enum otherschema.enum_type NOT NULL
);

-- test associations
CREATE TABLE otherschema.parents (
    id SERIAL PRIMARY KEY,
    value text NOT NULL
);
CREATE TABLE otherschema.children (
    id SERIAL PRIMARY KEY,
    value text NOT NULL,
    parent_id integer NOT NULL
        REFERENCES otherschema.parents(id) ON DELETE RESTRICT ON UPDATE CASCADE
);
CREATE TABLE otherschema.unconstrained_children (
    id SERIAL PRIMARY KEY,
    value text NOT NULL,
    parent_id integer NOT NULL -- will have to be explicitly configured
);

-- test cross-schema associations
CREATE TABLE otherschema.small_entity_children (
    id SERIAL PRIMARY KEY,
    value text NOT NULL,
    small_entity_id integer NOT NULL
        REFERENCES small_entities(id) ON DELETE RESTRICT ON UPDATE CASCADE
);
CREATE TABLE children_of_otherschema ( -- NOTE: created in public schema not otherschema
    id SERIAL PRIMARY KEY,
    value text NOT NULL,
    otherschema_parent_id integer NOT NULL
        REFERENCES otherschema.parents(id) ON DELETE RESTRICT ON UPDATE CASCADE
);

CREATE TABLE otherschema."funky ""name" (
    id SERIAL PRIMARY KEY,
    value text NOT NULL
);

