--
-- PostgreSQL database dump
--

-- Dumped from database version 12.1
-- Dumped by pg_dump version 12.1

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: uuid-ossp; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS "uuid-ossp" WITH SCHEMA public;


--
-- Name: EXTENSION "uuid-ossp"; Type: COMMENT; Schema: -; Owner: -
--

COMMENT ON EXTENSION "uuid-ossp" IS 'generate universally unique identifiers (UUIDs)';


--
-- Name: enum_type; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.enum_type AS ENUM (
    'option1',
    'option2'
);


--
-- Name: concats_text(text, text); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.concats_text(t1 text, t2 text) RETURNS text
    LANGUAGE plpgsql
    AS $$
BEGIN
    RETURN t1 || t2;
END
$$;


--
-- Name: get_small_entity_1(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.get_small_entity_1() RETURNS TABLE(id integer, anint integer)
    LANGUAGE sql
    AS $$
SELECT * FROM small_entities
$$;


--
-- Name: get_small_entity_2(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.get_small_entity_2() RETURNS TABLE(id integer, anint integer)
    LANGUAGE sql
    AS $$
SELECT * FROM small_entities
$$;


--
-- Name: returns_text(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.returns_text() RETURNS text
    LANGUAGE plpgsql
    AS $$
BEGIN
    RETURN 'foo';
END
$$;


--
-- Name: select_blobs(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.select_blobs() RETURNS TABLE(bytea_field bytea, bytea_field_not_null bytea)
    LANGUAGE sql
    AS $$
SELECT bytea_field, bytea_field_not_null FROM type_rainbow;
$$;


--
-- Name: select_bool(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.select_bool() RETURNS TABLE(bool_field boolean, bool_field_not_null boolean)
    LANGUAGE sql
    AS $$
SELECT bool_field, bool_field_not_null FROM type_rainbow;
$$;


--
-- Name: select_enum(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.select_enum() RETURNS TABLE(enum_field public.enum_type, enum_field_not_null public.enum_type)
    LANGUAGE sql
    AS $$
SELECT enum_field, enum_field_not_null FROM type_rainbow;
$$;


--
-- Name: select_matching_string(text); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.select_matching_string(must_be text) RETURNS TABLE(text_field text)
    LANGUAGE sql
    AS $$
SELECT t.text_field
FROM type_rainbow t
WHERE t.text_field = must_be;
$$;


--
-- Name: select_money(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.select_money() RETURNS TABLE(money_field money, money_field_not_null money)
    LANGUAGE sql
    AS $$
SELECT money_field, money_field_not_null from type_rainbow;
$$;


--
-- Name: select_numbers(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.select_numbers() RETURNS TABLE(smallint_field smallint, smallint_field_not_null smallint, integer_field integer, integer_field_not_null integer, bigint_field bigint, bigint_field_not_null bigint, decimal_field numeric, decimal_field_not_null numeric, numeric_field numeric, numeric_field_not_null numeric, numeric_prec_field numeric, numeric_prec_field_not_null numeric, numeric_prec_scale_field numeric, numeric_prec_scale_field_not_null numeric, real_field real, real_field_not_null real, double_field double precision, double_field_not_null double precision, serial_field integer, serial_field_not_null integer, big_serial_field bigint, big_serial_field_not_null bigint)
    LANGUAGE sql
    AS $$
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
$$;


--
-- Name: select_string_types(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.select_string_types() RETURNS TABLE(text_field text, text_field_not_null text, varchar_field character varying, varchar_field_not_null character varying, char_field character, char_field_not_null character)
    LANGUAGE plpgsql
    AS $$
BEGIN
    RETURN QUERY
    SELECT t.text_field, t.text_field_not_null,
           t.varchar_field, t.varchar_field_not_null,
           t.char_field, t.char_field_not_null
    FROM type_rainbow t;
END
$$;


--
-- Name: select_time(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.select_time() RETURNS TABLE(ts_field timestamp without time zone, ts_field_not_null timestamp without time zone, tsz_field timestamp with time zone, tsz_field_not_null timestamp with time zone, date_field date, date_field_not_null date, time_field time without time zone, time_field_not_null time without time zone, timez_field time with time zone, timez_field_not_null time with time zone)
    LANGUAGE sql
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
$$;


--
-- Name: select_uuid(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.select_uuid() RETURNS TABLE(uuid_field uuid, uuid_field_not_null uuid)
    LANGUAGE sql
    AS $$
SELECT uuid_field, uuid_field_not_null FROM type_rainbow;
$$;


SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: attachments; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.attachments (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    small_entity_id integer NOT NULL,
    value text
);


--
-- Name: schema_migrations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.schema_migrations (
    id text NOT NULL,
    applied_at timestamp with time zone
);


--
-- Name: small_entities; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.small_entities (
    id integer NOT NULL,
    anint integer NOT NULL
);


--
-- Name: small_entities_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.small_entities_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: small_entities_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.small_entities_id_seq OWNED BY public.small_entities.id;


--
-- Name: type_rainbow; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.type_rainbow (
    id integer NOT NULL,
    text_field text,
    text_field_not_null text NOT NULL,
    varchar_field character varying(40),
    varchar_field_not_null character varying(40) NOT NULL,
    char_field character(40),
    char_field_not_null character(40) NOT NULL,
    money_field money,
    money_field_not_null money NOT NULL,
    ts_field timestamp without time zone,
    ts_field_not_null timestamp without time zone NOT NULL,
    tsz_field timestamp with time zone,
    tsz_field_not_null timestamp with time zone NOT NULL,
    date_field date,
    date_field_not_null date NOT NULL,
    time_field time without time zone,
    time_field_not_null time without time zone NOT NULL,
    timez_field time with time zone,
    timez_field_not_null time with time zone NOT NULL,
    interval_field interval,
    interval_field_not_null interval NOT NULL,
    bool_field boolean,
    bool_field_not_null boolean NOT NULL,
    enum_field public.enum_type,
    enum_field_not_null public.enum_type NOT NULL,
    uuid_field uuid,
    uuid_field_not_null uuid NOT NULL,
    bytea_field bytea,
    bytea_field_not_null bytea NOT NULL,
    smallint_field smallint,
    smallint_field_not_null smallint NOT NULL,
    integer_field integer,
    integer_field_not_null integer NOT NULL,
    bigint_field bigint,
    bigint_field_not_null bigint NOT NULL,
    decimal_field numeric,
    decimal_field_not_null numeric NOT NULL,
    numeric_field numeric,
    numeric_field_not_null numeric NOT NULL,
    numeric_prec_field numeric(10,0),
    numeric_prec_field_not_null numeric(10,0) NOT NULL,
    numeric_prec_scale_field numeric(10,2),
    numeric_prec_scale_field_not_null numeric(10,2) NOT NULL,
    real_field real,
    real_field_not_null real NOT NULL,
    double_field double precision,
    double_field_not_null double precision NOT NULL,
    serial_field integer NOT NULL,
    serial_field_not_null integer NOT NULL,
    big_serial_field bigint NOT NULL,
    big_serial_field_not_null bigint NOT NULL
);


--
-- Name: type_rainbow_big_serial_field_not_null_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.type_rainbow_big_serial_field_not_null_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: type_rainbow_big_serial_field_not_null_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.type_rainbow_big_serial_field_not_null_seq OWNED BY public.type_rainbow.big_serial_field_not_null;


--
-- Name: type_rainbow_big_serial_field_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.type_rainbow_big_serial_field_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: type_rainbow_big_serial_field_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.type_rainbow_big_serial_field_seq OWNED BY public.type_rainbow.big_serial_field;


--
-- Name: type_rainbow_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.type_rainbow_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: type_rainbow_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.type_rainbow_id_seq OWNED BY public.type_rainbow.id;


--
-- Name: type_rainbow_serial_field_not_null_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.type_rainbow_serial_field_not_null_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: type_rainbow_serial_field_not_null_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.type_rainbow_serial_field_not_null_seq OWNED BY public.type_rainbow.serial_field_not_null;


--
-- Name: type_rainbow_serial_field_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.type_rainbow_serial_field_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: type_rainbow_serial_field_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.type_rainbow_serial_field_seq OWNED BY public.type_rainbow.serial_field;


--
-- Name: small_entities id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.small_entities ALTER COLUMN id SET DEFAULT nextval('public.small_entities_id_seq'::regclass);


--
-- Name: type_rainbow id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.type_rainbow ALTER COLUMN id SET DEFAULT nextval('public.type_rainbow_id_seq'::regclass);


--
-- Name: type_rainbow serial_field; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.type_rainbow ALTER COLUMN serial_field SET DEFAULT nextval('public.type_rainbow_serial_field_seq'::regclass);


--
-- Name: type_rainbow serial_field_not_null; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.type_rainbow ALTER COLUMN serial_field_not_null SET DEFAULT nextval('public.type_rainbow_serial_field_not_null_seq'::regclass);


--
-- Name: type_rainbow big_serial_field; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.type_rainbow ALTER COLUMN big_serial_field SET DEFAULT nextval('public.type_rainbow_big_serial_field_seq'::regclass);


--
-- Name: type_rainbow big_serial_field_not_null; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.type_rainbow ALTER COLUMN big_serial_field_not_null SET DEFAULT nextval('public.type_rainbow_big_serial_field_not_null_seq'::regclass);


--
-- Name: attachments attachments_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.attachments
    ADD CONSTRAINT attachments_pkey PRIMARY KEY (id);


--
-- Name: schema_migrations schema_migrations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.schema_migrations
    ADD CONSTRAINT schema_migrations_pkey PRIMARY KEY (id);


--
-- Name: small_entities small_entities_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.small_entities
    ADD CONSTRAINT small_entities_pkey PRIMARY KEY (id);


--
-- Name: type_rainbow type_rainbow_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.type_rainbow
    ADD CONSTRAINT type_rainbow_pkey PRIMARY KEY (id);


--
-- Name: attachments attachments_small_entity_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.attachments
    ADD CONSTRAINT attachments_small_entity_id_fkey FOREIGN KEY (small_entity_id) REFERENCES public.small_entities(id) ON UPDATE CASCADE ON DELETE RESTRICT;


--
-- PostgreSQL database dump complete
--

