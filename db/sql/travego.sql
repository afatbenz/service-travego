--
-- PostgreSQL database dump
--

-- Dumped from database version 16.9
-- Dumped by pg_dump version 16.9

-- Started on 2026-06-29 00:49:33

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

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- TOC entry 275 (class 1259 OID 33985)
-- Name: _assistant; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public._assistant (
    account_id uuid,
    user_id uuid,
    username character varying(50),
    phone_number character varying(20),
    status integer,
    created_at timestamp with time zone,
    updated_at timestamp with time zone
);


ALTER TABLE public._assistant OWNER TO postgres;

--
-- TOC entry 276 (class 1259 OID 33988)
-- Name: _packages; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public._packages (
    package_id character varying(10),
    package_name character varying(20),
    package_price numeric,
    original_price numeric,
    fleet_limit integer,
    tour_package_limit integer,
    fleet_order_limit integer,
    tour_order_limit integer,
    assistant_account_limit integer,
    assistant_request_limit numeric
);


ALTER TABLE public._packages OWNER TO postgres;

--
-- TOC entry 273 (class 1259 OID 33977)
-- Name: _subscription; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public._subscription (
    subscription_id uuid,
    organization_id uuid,
    package_id character varying(10),
    activate_date date,
    expiry_date date,
    subscription_type integer,
    status integer,
    package_price numeric,
    created_at timestamp with time zone,
    updated_at timestamp with time zone
);


ALTER TABLE public._subscription OWNER TO postgres;

--
-- TOC entry 274 (class 1259 OID 33980)
-- Name: _subscription_payment; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public._subscription_payment (
    payment_id uuid,
    invoice_id character varying,
    subscription_id uuid,
    user_id uuid,
    payment_amount numeric,
    discount numeric,
    promotion_id uuid,
    referral_id uuid,
    payment_type character varying(20),
    payment_date timestamp with time zone,
    merchant_id character varying(20)
);


ALTER TABLE public._subscription_payment OWNER TO postgres;

--
-- TOC entry 277 (class 1259 OID 33993)
-- Name: _usage; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public._usage (
    usage_id uuid,
    subscription_id uuid,
    user_id uuid,
    fleet_limit integer,
    tour_package_limit integer,
    fleet_order_limit integer,
    tour_order_limit integer,
    assistant_limit integer,
    created_at timestamp with time zone
);


ALTER TABLE public._usage OWNER TO postgres;

--
-- TOC entry 310 (class 1259 OID 51594)
-- Name: assistant_account_stats; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.assistant_account_stats (
    statistic_id uuid,
    period date,
    count integer,
    organization_id uuid,
    type integer,
    status integer
);


ALTER TABLE public.assistant_account_stats OWNER TO postgres;

--
-- TOC entry 287 (class 1259 OID 35036)
-- Name: assistant_accounts; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.assistant_accounts (
    assistant_id uuid,
    organization_id uuid,
    user_type integer,
    user_id uuid,
    account_number character varying(17),
    account_name character varying(50),
    created_at timestamp with time zone,
    created_by uuid,
    status integer
);


ALTER TABLE public.assistant_accounts OWNER TO postgres;

--
-- TOC entry 309 (class 1259 OID 51589)
-- Name: assistant_customer_stats; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.assistant_customer_stats (
    statistic_id uuid,
    period date,
    count integer,
    organization_id uuid,
    type integer,
    status integer
);


ALTER TABLE public.assistant_customer_stats OWNER TO postgres;

--
-- TOC entry 296 (class 1259 OID 51486)
-- Name: assistant_customers; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.assistant_customers (
    device_id character varying(150),
    device_name character varying(50),
    assistant_device_id character varying(40),
    account character varying(20),
    device_token character varying(200),
    organization_id uuid,
    created_at timestamp with time zone,
    created_by uuid,
    updated_at timestamp with time zone,
    updated_by uuid
);


ALTER TABLE public.assistant_customers OWNER TO postgres;

--
-- TOC entry 235 (class 1259 OID 25581)
-- Name: bank_list; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.bank_list (
    code character varying(10) NOT NULL,
    name text NOT NULL,
    icon character varying(100)
);


ALTER TABLE public.bank_list OWNER TO postgres;

--
-- TOC entry 227 (class 1259 OID 17407)
-- Name: content; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.content (
    uuid uuid,
    section_tag character varying(100),
    organization_id uuid,
    content text,
    parent character varying(100),
    is_active boolean,
    type character varying(20),
    fuel_type character varying(10),
    transmission character varying(20),
    created_at timestamp with time zone,
    created_by uuid,
    updated_by uuid,
    updated_at timestamp with time zone
);


ALTER TABLE public.content OWNER TO postgres;

--
-- TOC entry 228 (class 1259 OID 17412)
-- Name: content_list; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.content_list (
    uuid uuid,
    content_id uuid,
    icon character varying,
    label character varying(100),
    sub_label character varying(255),
    created_at timestamp with time zone,
    updated_at timestamp with time zone
);


ALTER TABLE public.content_list OWNER TO postgres;

--
-- TOC entry 250 (class 1259 OID 33862)
-- Name: customer_orders; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.customer_orders (
    order_id character varying(100),
    customer_id uuid,
    order_type integer,
    created_at timestamp with time zone,
    created_by uuid,
    updated_at timestamp with time zone,
    updated_by uuid,
    organization_id uuid
);


ALTER TABLE public.customer_orders OWNER TO postgres;

--
-- TOC entry 249 (class 1259 OID 33857)
-- Name: customers; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.customers (
    customer_id uuid,
    organization_id uuid,
    customer_name character varying(100),
    customer_email character varying(100),
    customer_address character varying(100),
    customer_city integer,
    customer_phone character varying(16),
    customer_bod date,
    customer_company character varying(100),
    created_at timestamp with time zone,
    created_by uuid,
    updated_at timestamp with time zone,
    updated_by uuid
);


ALTER TABLE public.customers OWNER TO postgres;

--
-- TOC entry 255 (class 1259 OID 33882)
-- Name: employee; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.employee (
    uuid uuid,
    employee_id character varying(100),
    nik character varying(20),
    fullname character varying(50),
    phone character varying(20),
    birth_date date,
    email character varying(50),
    address character varying(255),
    address_city integer,
    join_date date,
    role_id uuid,
    organization_id uuid,
    avatar character varying(200),
    contract_status integer,
    status integer,
    resign_date date,
    created_at timestamp with time zone,
    created_by uuid,
    updated_at timestamp with time zone,
    updated_by uuid
);


ALTER TABLE public.employee OWNER TO postgres;

--
-- TOC entry 266 (class 1259 OID 33927)
-- Name: employee_leave_type; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.employee_leave_type (
    id integer,
    label character varying(50)
);


ALTER TABLE public.employee_leave_type OWNER TO postgres;

--
-- TOC entry 265 (class 1259 OID 33924)
-- Name: employee_leaves; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.employee_leaves (
    leave_id uuid,
    organization_id uuid,
    employee_id uuid,
    substituted_by uuid,
    start_date date,
    end_date date,
    leave_type integer,
    status integer,
    created_at timestamp with time zone,
    created_by uuid,
    updated_at timestamp with time zone,
    updated_by uuid
);


ALTER TABLE public.employee_leaves OWNER TO postgres;

--
-- TOC entry 264 (class 1259 OID 33921)
-- Name: employee_shift; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.employee_shift (
    shift_id uuid,
    organization_id uuid,
    employee_id uuid,
    shift_date date,
    shift_type integer,
    created_at timestamp with time zone,
    created_by uuid,
    updated_at timestamp with time zone,
    updated_by uuid
);


ALTER TABLE public.employee_shift OWNER TO postgres;

--
-- TOC entry 299 (class 1259 OID 51513)
-- Name: facilities; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.facilities (
    facility_id uuid,
    facility_name character varying(200),
    facility_icon character(20),
    organization_id uuid
);


ALTER TABLE public.facilities OWNER TO postgres;

--
-- TOC entry 224 (class 1259 OID 17357)
-- Name: fleet_addon; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.fleet_addon (
    uuid uuid,
    fleet_id uuid,
    addon_name character varying(255),
    addon_desc text,
    addon_price integer,
    created_at timestamp with time zone,
    created_by uuid,
    updated_at timestamp with time zone,
    updated_by uuid,
    organization_id uuid
);


ALTER TABLE public.fleet_addon OWNER TO postgres;

--
-- TOC entry 300 (class 1259 OID 51517)
-- Name: fleet_facilities; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.fleet_facilities (
    uuid uuid,
    fleet_id uuid,
    facility_id uuid,
    created_by uuid,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    updated_by uuid,
    organization_id uuid
);


ALTER TABLE public.fleet_facilities OWNER TO postgres;

--
-- TOC entry 226 (class 1259 OID 17365)
-- Name: fleet_images; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.fleet_images (
    uuid uuid,
    fleet_id uuid,
    path_file character varying(255),
    organization_id uuid
);


ALTER TABLE public.fleet_images OWNER TO postgres;

--
-- TOC entry 231 (class 1259 OID 25560)
-- Name: fleet_order_addons; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.fleet_order_addons (
    order_addon_id uuid,
    order_id character varying(50),
    addon_id uuid,
    addon_price numeric,
    organization_id uuid,
    order_item_id uuid,
    created_at timestamp with time zone
);


ALTER TABLE public.fleet_order_addons OWNER TO postgres;

--
-- TOC entry 233 (class 1259 OID 25568)
-- Name: fleet_order_customers; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.fleet_order_customers (
    customer_id uuid,
    order_id character varying(50),
    customer_name character varying(100),
    customer_phone character varying(20),
    customer_email character varying(50),
    customer_address character varying(255),
    organization_id uuid,
    created_at timestamp with time zone
);


ALTER TABLE public.fleet_order_customers OWNER TO postgres;

--
-- TOC entry 232 (class 1259 OID 25565)
-- Name: fleet_order_destinations; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.fleet_order_destinations (
    order_destination_id uuid,
    order_id character varying(50),
    city_id integer,
    location character varying(255),
    created_at timestamp with time zone
);


ALTER TABLE public.fleet_order_destinations OWNER TO postgres;

--
-- TOC entry 283 (class 1259 OID 34061)
-- Name: fleet_order_expenses; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.fleet_order_expenses (
    fleet_expense_id uuid,
    expense_id uuid,
    schedule_id uuid,
    trip_id character varying(50),
    amount numeric,
    quantity integer,
    total_amount numeric,
    payment_type integer,
    organization_id uuid,
    created_at timestamp with time zone,
    created_by uuid,
    updated_at timestamp with time zone,
    updated_by uuid
);


ALTER TABLE public.fleet_order_expenses OWNER TO postgres;

--
-- TOC entry 260 (class 1259 OID 33901)
-- Name: fleet_order_items; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.fleet_order_items (
    order_item_id uuid,
    organization_id uuid,
    order_id character varying(100),
    fleet_id uuid,
    price_id uuid,
    quantity numeric,
    charge_amount numeric,
    discount numeric,
    sub_total numeric,
    addon_amount numeric,
    status integer,
    create_at timestamp with time zone,
    created_by uuid,
    updated_at timestamp with time zone,
    updated_by uuid
);


ALTER TABLE public.fleet_order_items OWNER TO postgres;

--
-- TOC entry 251 (class 1259 OID 33865)
-- Name: fleet_order_itinerary; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.fleet_order_itinerary (
    fleet_itinerary_id uuid,
    order_id character varying(100),
    day_num integer,
    city_id integer,
    location character varying(100),
    organization_id uuid,
    created_at timestamp with time zone,
    created_by uuid,
    updated_at timestamp with time zone,
    updated_by uuid
);


ALTER TABLE public.fleet_order_itinerary OWNER TO postgres;

--
-- TOC entry 236 (class 1259 OID 25591)
-- Name: fleet_order_payment; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.fleet_order_payment (
    order_payment_id uuid,
    order_id character varying(50),
    organization_id uuid,
    payment_method uuid,
    payment_type integer,
    payment_percentage integer,
    payment_amount numeric,
    total_amount numeric,
    payment_remaining numeric,
    unique_code character varying(10),
    evidence_file character varying(100),
    status integer,
    created_at timestamp with time zone,
    settled_at timestamp with time zone,
    canceled_at timestamp with time zone,
    approve_by uuid
);


ALTER TABLE public.fleet_order_payment OWNER TO postgres;

--
-- TOC entry 230 (class 1259 OID 17444)
-- Name: fleet_orders; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.fleet_orders (
    order_id character varying(50),
    fleet_id uuid,
    start_date timestamp with time zone,
    end_date timestamp with time zone,
    pickup_city_id integer,
    pickup_location character varying(255),
    unit_qty integer,
    price_id uuid,
    total_amount numeric,
    additional_amount numeric,
    discount numeric,
    additional_request text,
    status integer,
    payment_status integer,
    organization_id uuid,
    created_at timestamp with time zone,
    created_by uuid,
    approve_by uuid,
    approve_date timestamp with time zone,
    cancel_by uuid,
    cancel_date timestamp with time zone,
    updated_at timestamp with time zone,
    updated_by uuid
);


ALTER TABLE public.fleet_orders OWNER TO postgres;

--
-- TOC entry 222 (class 1259 OID 17351)
-- Name: fleet_pickup; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.fleet_pickup (
    uuid uuid,
    fleet_id uuid,
    city_id integer,
    created_at timestamp with time zone,
    created_by uuid,
    updated_at timestamp with time zone,
    updated_by uuid,
    organization_id uuid
);


ALTER TABLE public.fleet_pickup OWNER TO postgres;

--
-- TOC entry 223 (class 1259 OID 17354)
-- Name: fleet_prices; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.fleet_prices (
    uuid uuid,
    fleet_id uuid,
    duration integer,
    rent_type integer,
    price integer,
    disc_amount integer,
    disc_price integer,
    uom character varying(10),
    created_by uuid,
    created_at timestamp with time zone,
    updated_by uuid,
    updated_at timestamp with time zone,
    organization_id uuid
);


ALTER TABLE public.fleet_prices OWNER TO postgres;

--
-- TOC entry 239 (class 1259 OID 25612)
-- Name: fleet_prices_history; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.fleet_prices_history (
    uuid uuid,
    fleet_id uuid,
    duration integer,
    rent_type integer,
    price integer,
    disc_amount integer,
    disc_price integer,
    uom character varying(10),
    created_by uuid,
    created_at timestamp with time zone,
    updated_by uuid,
    updated_at timestamp with time zone,
    organization_id uuid
);


ALTER TABLE public.fleet_prices_history OWNER TO postgres;

--
-- TOC entry 225 (class 1259 OID 17362)
-- Name: fleet_types; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.fleet_types (
    id character varying(5),
    label character varying(50)
);


ALTER TABLE public.fleet_types OWNER TO postgres;

--
-- TOC entry 278 (class 1259 OID 33996)
-- Name: fleet_unit_ownership; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.fleet_unit_ownership (
    fleet_ownership_id uuid,
    unit_id uuid,
    partner_id uuid,
    created_at timestamp with time zone,
    created_by uuid,
    updated_at timestamp with time zone,
    updated_by uuid,
    organization_id uuid
);


ALTER TABLE public.fleet_unit_ownership OWNER TO postgres;

--
-- TOC entry 252 (class 1259 OID 33868)
-- Name: fleet_units; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.fleet_units (
    unit_id uuid,
    vehicle_id character varying(100),
    plate_number character varying(20),
    fleet_id uuid,
    engine character varying(100),
    capacity integer,
    production_year integer,
    transmission character varying(20),
    ownership_type integer,
    organization_id uuid,
    status integer,
    created_at timestamp with time zone,
    created_by uuid,
    updated_at timestamp with time zone,
    updated_by uuid
);


ALTER TABLE public.fleet_units OWNER TO postgres;

--
-- TOC entry 221 (class 1259 OID 17343)
-- Name: fleets; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.fleets (
    uuid uuid,
    fleet_name character varying(100),
    fleet_type character varying(5),
    capacity integer,
    production_year integer,
    engine character varying(50),
    body character varying(50),
    description text,
    organization_id uuid,
    thumbnail character varying(255),
    fuel_type character varying(10),
    transmission character varying(20),
    views integer,
    is_public integer,
    active boolean,
    status integer,
    created_at timestamp with time zone,
    created_by uuid,
    updated_at timestamp with time zone,
    updated_by uuid
);


ALTER TABLE public.fleets OWNER TO postgres;

--
-- TOC entry 290 (class 1259 OID 43276)
-- Name: garage; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.garage (
    organization_id uuid,
    garage_id uuid,
    garage_name character varying(50),
    garage_address character varying(255),
    garage_city character varying(50),
    status integer,
    created_at timestamp with time zone,
    created_by uuid,
    updated_at timestamp with time zone,
    updated_by uuid
);


ALTER TABLE public.garage OWNER TO postgres;

--
-- TOC entry 229 (class 1259 OID 17417)
-- Name: hot_offers; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.hot_offers (
    promo_id uuid,
    service_type character varying(10),
    product_id uuid,
    discount_type character varying(10),
    discount_value bigint,
    period_start timestamp with time zone,
    period_end timestamp with time zone,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    organization_id uuid,
    created_by uuid,
    updated_by uuid
);


ALTER TABLE public.hot_offers OWNER TO postgres;

--
-- TOC entry 292 (class 1259 OID 43285)
-- Name: inventory_item_garage; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.inventory_item_garage (
    item_garage_id uuid,
    item_id uuid,
    garage_id uuid,
    stock integer,
    organization_id uuid,
    created_at timestamp with time zone,
    created_by uuid,
    updated_at timestamp with time zone,
    updated_by uuid
);


ALTER TABLE public.inventory_item_garage OWNER TO postgres;

--
-- TOC entry 297 (class 1259 OID 51489)
-- Name: inventory_item_supliers; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.inventory_item_supliers (
    item_id uuid,
    suplier_id uuid,
    transaction_id uuid,
    created_at timestamp with time zone,
    created_by uuid
);


ALTER TABLE public.inventory_item_supliers OWNER TO postgres;

--
-- TOC entry 291 (class 1259 OID 43282)
-- Name: inventory_items; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.inventory_items (
    item_id uuid,
    organization_id uuid,
    item_name character varying(100),
    item_uom character varying(20),
    item_category integer,
    stock integer,
    item_sku character varying(20),
    item_price numeric,
    status integer,
    created_at timestamp with time zone,
    created_by uuid,
    updated_at timestamp with time zone,
    updated_by uuid
);


ALTER TABLE public.inventory_items OWNER TO postgres;

--
-- TOC entry 294 (class 1259 OID 43307)
-- Name: inventory_movement; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.inventory_movement (
    movement_id uuid,
    item_id uuid,
    garage_id uuid,
    quantity integer,
    stock_before integer,
    stock_final integer,
    movement_type integer,
    created_at timestamp with time zone,
    created_by uuid,
    organization_id uuid,
    notes character varying(100)
);


ALTER TABLE public.inventory_movement OWNER TO postgres;

--
-- TOC entry 295 (class 1259 OID 43310)
-- Name: inventory_movement_types; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.inventory_movement_types (
    id integer,
    label character varying
);


ALTER TABLE public.inventory_movement_types OWNER TO postgres;

--
-- TOC entry 298 (class 1259 OID 51505)
-- Name: inventory_orders; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.inventory_orders (
    request_id character varying(20),
    purchase_id character varying(20),
    item_id uuid,
    garage_id uuid,
    suplier_id uuid,
    quantity integer,
    item_price numeric,
    total_amount numeric,
    item_category integer,
    transaction_date date,
    organization_id uuid,
    status integer,
    complete_date date,
    created_at timestamp with time zone,
    created_by uuid,
    updated_at timestamp with time zone,
    updated_by uuid
);


ALTER TABLE public.inventory_orders OWNER TO postgres;

--
-- TOC entry 302 (class 1259 OID 51547)
-- Name: inventory_request; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.inventory_request (
    request_id character varying(50),
    item_category integer,
    item_id uuid,
    item_name character varying(50),
    item_uom character varying(10),
    garage_id uuid,
    employee_id uuid,
    quantity integer,
    notes text,
    status integer,
    created_at timestamp with time zone,
    created_by uuid,
    approve_at timestamp with time zone,
    approve_by uuid,
    updated_at timestamp with time zone,
    updated_by uuid,
    organization_id uuid,
    received_at timestamp with time zone,
    received_by uuid
);


ALTER TABLE public.inventory_request OWNER TO postgres;

--
-- TOC entry 301 (class 1259 OID 51539)
-- Name: inventory_request_fleets; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.inventory_request_fleets (
    request_id character varying(50),
    unit_id uuid,
    created_at timestamp with time zone,
    created_by uuid,
    updated_at timestamp with time zone,
    updated_by uuid
);


ALTER TABLE public.inventory_request_fleets OWNER TO postgres;

--
-- TOC entry 271 (class 1259 OID 33951)
-- Name: messages; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.messages (
    message_id uuid,
    customer_name character varying(100),
    customer_email character varying(50),
    customer_phone character varying(20),
    message_type character varying(20),
    message text,
    status integer,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    organization_id uuid
);


ALTER TABLE public.messages OWNER TO postgres;

--
-- TOC entry 286 (class 1259 OID 35031)
-- Name: notifications; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.notifications (
    notification_id uuid,
    organization_id uuid,
    reference_url text,
    title character varying(50),
    message character varying(100),
    created_at timestamp with time zone,
    is_read boolean
);


ALTER TABLE public.notifications OWNER TO postgres;

--
-- TOC entry 279 (class 1259 OID 33999)
-- Name: operation_partner; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.operation_partner (
    partner_id uuid,
    partner_name character varying(50),
    partner_address character varying(100),
    partner_city integer,
    partner_phone character varying(20),
    pic_name character varying(50),
    partner_email character varying(50),
    created_at timestamp with time zone,
    created_by uuid,
    updated_at timestamp with time zone,
    updated_by uuid,
    organization_id uuid
);


ALTER TABLE public.operation_partner OWNER TO postgres;

--
-- TOC entry 237 (class 1259 OID 25596)
-- Name: order_payment_history; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.order_payment_history (
    payment_history_id uuid,
    order_id character varying(50),
    bank_account_id uuid,
    bank_code character varying(10),
    account_number character varying(30),
    account_name character varying(50),
    payment_amount numeric,
    unique_code character varying(10),  
    organization_id uuid,
    created_at timestamp with time zone
);


ALTER TABLE public.order_payment_history OWNER TO postgres;

--
-- TOC entry 272 (class 1259 OID 33961)
-- Name: order_reviews; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.order_reviews (
    review_id uuid,
    star integer,
    review text,
    organization_id uuid,
    customer_id uuid,
    order_type integer,
    order_id character varying(50),
    created_at timestamp with time zone
);


ALTER TABLE public.order_reviews OWNER TO postgres;

--
-- TOC entry 234 (class 1259 OID 25576)
-- Name: organization_bank_accounts; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.organization_bank_accounts (
    bank_account_id uuid,
    bank_code character varying(10),
    account_number character varying(30),
    account_name character varying(50),
    merchant_id character varying(50),
    merchant_nmid character varying(50),
    merchant_name character varying(150),
    merchant_mcc character varying(50),
    merchant_address character varying(255),
    merchant_city integer,
    merchant_postal_code character varying(10),
    account_type integer,
    organization_id uuid,
    status integer,
    created_at timestamp with time zone,
    created_by uuid,
    updated_by uuid,
    updated_at timestamp with time zone,
    created_proxy character varying(50),
    updated_proxy character varying(50),
    created_ip character varying(50),
    updated_ip character varying(50),
    active boolean
);


ALTER TABLE public.organization_bank_accounts OWNER TO postgres;

--
-- TOC entry 254 (class 1259 OID 33879)
-- Name: organization_divisions; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.organization_divisions (
    division_id uuid,
    division_name character varying(100),
    description character varying(255),
    organization_id uuid,
    created_at timestamp with time zone,
    created_by uuid,
    updated_at timestamp with time zone,
    updated_by uuid,
    status integer
);


ALTER TABLE public.organization_divisions OWNER TO postgres;

--
-- TOC entry 238 (class 1259 OID 25609)
-- Name: organization_members; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.organization_members (
    member_id uuid,
    fullname character varying(50),
    nip character varying(50),
    nik character varying(16),
    phone character varying(20),
    email character varying(50),
    division_id uuid,
    "position" character varying(50),
    npwp character varying(30),
    bank_code character varying(10),
    bank_account_number character varying(20),
    bank_account_name character varying(50),
    organization_id uuid,
    created_at timestamp with time zone,
    created_by uuid,
    updated_at timestamp with time zone,
    updated_by uuid,
    active boolean
);


ALTER TABLE public.organization_members OWNER TO postgres;

--
-- TOC entry 253 (class 1259 OID 33876)
-- Name: organization_roles; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.organization_roles (
    role_id uuid,
    description character varying(255),
    role_name character varying(100),
    division_id uuid,
    organization_id uuid,
    created_at timestamp with time zone,
    created_by uuid,
    updated_at timestamp with time zone,
    updated_by uuid,
    status integer
);


ALTER TABLE public.organization_roles OWNER TO postgres;

--
-- TOC entry 220 (class 1259 OID 17312)
-- Name: organization_types; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.organization_types (
    id integer,
    name character varying(50)
);


ALTER TABLE public.organization_types OWNER TO postgres;

--
-- TOC entry 218 (class 1259 OID 17257)
-- Name: organization_users; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.organization_users (
    uuid uuid NOT NULL,
    user_id uuid NOT NULL,
    organization_id uuid NOT NULL,
    organization_role integer NOT NULL,
    is_active boolean,
    created_at timestamp with time zone,
    created_by uuid NOT NULL,
    updated_at timestamp with time zone,
    updated_by uuid NOT NULL
);


ALTER TABLE public.organization_users OWNER TO postgres;

--
-- TOC entry 217 (class 1259 OID 17241)
-- Name: organizations; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.organizations (
    organization_id uuid NOT NULL,
    organization_code character varying(10) NOT NULL,
    organization_name character varying(255) NOT NULL,
    company_name character varying(200),
    phone character varying(20),
    address character varying(100),
    city character varying(100),
    province character varying(30),
    npwp_number character varying(30),
    email character varying(50),
    created_by uuid NOT NULL,
    organization_type integer NOT NULL,
    postal_code character varying(10),
    organization_icon text,
    domain_url character varying(100),
    logo character varying(50),
    organization_lat character varying(200),
    organization_lng text,
    address_label character varying(50),
    whatsapp character varying(20),
    created_at timestamp with time zone,
    updated_at timestamp with time zone
);


ALTER TABLE public.organizations OWNER TO postgres;

--
-- TOC entry 270 (class 1259 OID 33946)
-- Name: payment_midtrans; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.payment_midtrans (
    transaction_id uuid,
    transaction_status character varying(20),
    order_id character varying(50),
    payment_type character varying(50),
    merchant_id character varying(50),
    gross_amount numeric,
    currency character varying(10),
    transaction_time timestamp without time zone,
    payment_status character varying(10),
    created_at timestamp with time zone
);


ALTER TABLE public.payment_midtrans OWNER TO postgres;

--
-- TOC entry 256 (class 1259 OID 33887)
-- Name: payment_orders; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.payment_orders (
    payment_id uuid,
    order_type integer,
    order_id character varying(50),
    organization_id uuid,
    transaction_id uuid,
    invoice_number character varying(50),
    payment_type integer,
    payment_method integer,
    bank_id character varying(10),
    bank_account character varying(100),
    payment_amount numeric,
    total_amount numeric,
    remaining_amount numeric,
    unique_code numeric,
    evidence_file character varying(255),
    notes character varying(100),
    status integer,
    payment_status integer,
    created_at timestamp with time zone,
    created_by uuid,
    updated_at timestamp with time zone,
    updated_by uuid,
    settled_at timestamp with time zone,
    settled_by uuid,
    refund_at timestamp with time zone,
    refund_by uuid
);


ALTER TABLE public.payment_orders OWNER TO postgres;

--
-- TOC entry 280 (class 1259 OID 34003)
-- Name: preference_cities; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.preference_cities (
    preference_id uuid,
    city_id integer,
    province_id integer,
    minimal_day integer,
    organization_id uuid,
    created_at timestamp with time zone,
    created_by uuid
);


ALTER TABLE public.preference_cities OWNER TO postgres;

--
-- TOC entry 281 (class 1259 OID 34006)
-- Name: preference_city_types; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.preference_city_types (
    preference_type_id uuid,
    city_id integer,
    service_type integer,
    organization_id uuid
);


ALTER TABLE public.preference_city_types OWNER TO postgres;

--
-- TOC entry 259 (class 1259 OID 33898)
-- Name: schedule_fleet_teams; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.schedule_fleet_teams (
    uuid uuid,
    schedule_id uuid,
    unit_id uuid,
    schedule_fleet_id uuid,
    driver_id uuid,
    crew_id uuid,
    organization_id uuid,
    status integer,
    created_at timestamp with time zone,
    created_by uuid,
    updated_at timestamp with time zone,
    updated_by uuid
);


ALTER TABLE public.schedule_fleet_teams OWNER TO postgres;

--
-- TOC entry 258 (class 1259 OID 33895)
-- Name: schedule_fleets; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.schedule_fleets (
    uuid uuid,
    schedule_id uuid,
    schedule_number character varying(20),
    order_id character varying(100),
    fleet_id uuid,
    unit_id uuid,
    status integer,
    created_at timestamp with time zone,
    created_by uuid,
    updated_at timestamp with time zone,
    updated_by uuid,
    departure_time time with time zone,
    organization_id uuid,
);


ALTER TABLE public.schedule_fleets OWNER TO postgres;

--
-- TOC entry 261 (class 1259 OID 33909)
-- Name: schedule_teams; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.schedule_teams (
    schedule_team_id uuid,
    employee_id uuid,
    order_id character varying(100),
    order_type integer,
    start_date timestamp with time zone,
    end_date timestamp with time zone,
    created_at timestamp with time zone,
    created_by uuid,
    updated_at timestamp with time zone,
    updated_by uuid,
    status integer,
    organization_id uuid
);


ALTER TABLE public.schedule_teams OWNER TO postgres;

--
-- TOC entry 257 (class 1259 OID 33892)
-- Name: schedules; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.schedules (
    schedule_id uuid,
    organization_id uuid,
    order_id character varying(100),
    order_type integer,
    departure_time timestamp with time zone,
    arrival_time timestamp with time zone,
    status integer,
    created_at timestamp with time zone,
    created_by uuid,
    updated_at timestamp with time zone,
    updated_by uuid
);


ALTER TABLE public.schedules OWNER TO postgres;

--
-- TOC entry 293 (class 1259 OID 43299)
-- Name: supliers; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.supliers (
    suplier_id uuid,
    suplier_name character varying(50),
    suplier_address character varying(200),
    suplier_city integer,
    suplier_phone character varying(20),
    supliter_email character varying(50),
    created_at timestamp with time zone,
    created_by uuid,
    updated_at timestamp with time zone,
    updated_by uuid,
    suplier_url character varying(100)
);


ALTER TABLE public.supliers OWNER TO postgres;

--
-- TOC entry 247 (class 1259 OID 25668)
-- Name: tour_package_addons; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.tour_package_addons (
    uuid uuid,
    package_id uuid,
    organization_id uuid,
    description character varying(255),
    price numeric,
    created_at timestamp with time zone,
    created_by uuid,
    updated_by uuid,
    uppdated_at timestamp with time zone
);


ALTER TABLE public.tour_package_addons OWNER TO postgres;

--
-- TOC entry 244 (class 1259 OID 25657)
-- Name: tour_package_destinations; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.tour_package_destinations (
    uuid uuid,
    package_id uuid,
    organization_id uuid,
    city_id integer,
    destination character varying(100),
    created_at timestamp with time zone,
    created_by uuid,
    updated_at timestamp with time zone,
    updated_by uuid
);


ALTER TABLE public.tour_package_destinations OWNER TO postgres;

--
-- TOC entry 242 (class 1259 OID 25650)
-- Name: tour_package_facilities; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.tour_package_facilities (
    uuid uuid,
    package_id uuid,
    organization_id uuid,
    facility character varying(255),
    created_by uuid,
    created_at timestamp with time zone,
    updated_by uuid,
    updated_at timestamp with time zone
);


ALTER TABLE public.tour_package_facilities OWNER TO postgres;

--
-- TOC entry 248 (class 1259 OID 25680)
-- Name: tour_package_images; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.tour_package_images (
    uuid uuid,
    package_id uuid,
    organization_id uuid,
    image_path text,
    created_at timestamp with time zone,
    created_by uuid
);


ALTER TABLE public.tour_package_images OWNER TO postgres;

--
-- TOC entry 245 (class 1259 OID 25660)
-- Name: tour_package_itineraries; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.tour_package_itineraries (
    uuid uuid,
    package_id uuid,
    organization_id uuid,
    dayx time with time zone,
    activity text,
    city_id integer,
    day integer,
    location character varying(100),
    created_at timestamp with time zone,
    created_by uuid,
    updated_at timestamp with time zone,
    updated_by uuid
);


ALTER TABLE public.tour_package_itineraries OWNER TO postgres;

--
-- TOC entry 263 (class 1259 OID 33917)
-- Name: tour_package_order_addons; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.tour_package_order_addons (
    order_id character varying(100),
    organization_id uuid,
    addon_id uuid,
    created_at timestamp with time zone,
    created_by uuid,
    updated_at timestamp with time zone,
    updated_by uuid
);


ALTER TABLE public.tour_package_order_addons OWNER TO postgres;

--
-- TOC entry 262 (class 1259 OID 33912)
-- Name: tour_package_orders; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.tour_package_orders (
    uuid uuid,
    order_id character varying(100),
    tour_package_id uuid,
    customer_id uuid,
    start_date timestamp with time zone,
    end_date timestamp with time zone,
    total_pax integer,
    official_pax integer,
    member_pax integer,
    discount_amount numeric,
    additional_amount numeric,
    total_amount numeric,
    organization_id uuid,
    status integer,
    payment_status integer,
    pickup_address text,
    pickup_city_id integer,
    created_by uuid,
    created_at timestamp with time zone,
    updated_by uuid,
    updated_at timestamp with time zone
);


ALTER TABLE public.tour_package_orders OWNER TO postgres;

--
-- TOC entry 241 (class 1259 OID 25647)
-- Name: tour_package_pickup; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.tour_package_pickup (
    uuid uuid,
    package_id uuid,
    city_id integer,
    created_at timestamp with time zone,
    created_by uuid,
    updated_at timestamp with time zone,
    updated_by uuid,
    organization_id uuid
);


ALTER TABLE public.tour_package_pickup OWNER TO postgres;

--
-- TOC entry 243 (class 1259 OID 25653)
-- Name: tour_package_prices; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.tour_package_prices (
    uuid uuid,
    package_id uuid,
    organization_id uuid,
    min_pax integer,
    max_pax integer,
    price numeric,
    created_at timestamp with time zone,
    created_by uuid,
    updated_at timestamp with time zone,
    updated_by uuid
);


ALTER TABLE public.tour_package_prices OWNER TO postgres;

--
-- TOC entry 246 (class 1259 OID 25665)
-- Name: tour_package_schedules; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.tour_package_schedules (
    uuid uuid,
    package_id uuid,
    organization_id uuid,
    date_start date,
    date_end date,
    created_at timestamp with time zone,
    created_by uuid,
    updated_at timestamp with time zone,
    updated_by uuid,
    status integer,
    active integer
);


ALTER TABLE public.tour_package_schedules OWNER TO postgres;

--
-- TOC entry 240 (class 1259 OID 25642)
-- Name: tour_packages; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.tour_packages (
    uuid uuid,
    package_type integer,
    package_name character varying(100),
    package_description text,
    min_pax integer,
    max_pax integer,
    thumbnail character varying(255),
    duration integer,
    active boolean,
    status integer,
    organization_id uuid,
    created_at timestamp with time zone,
    created_by uuid,
    updated_at timestamp with time zone,
    updated_by uuid
);


ALTER TABLE public.tour_packages OWNER TO postgres;

--
-- TOC entry 284 (class 1259 OID 34977)
-- Name: transaction_fleet_trips; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.transaction_fleet_trips (
    transaction_trip_id uuid,
    transaction_id uuid,
    schedule_number character varying(50),
    reference_id character varying(50),
    transaction_date date
    transaction_type integer,
    transaction_category character varying(10),
    transaction_item character varying(10),
    amount numeric,
    payment_type integer,
    description text,
    status integer,
    organization_id uuid,
    created_at timestamp with time zone,
    created_by uuid,
    updated_at timestamp with time zone,
    updated_by uuid
);


ALTER TABLE public.transaction_fleet_trips OWNER TO postgres;

--
-- TOC entry 268 (class 1259 OID 33940)
-- Name: transaction_fleets; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.transaction_fleets (
    transaction_fleet_id uuid,
    transaction_id uuid,
    fleet_unit_id uuid,
    organization_id uuid,
    created_at timestamp with time zone,
    created_by uuid,
    updated_at timestamp with time zone,
    updated_by uuid
);


ALTER TABLE public.transaction_fleets OWNER TO postgres;

--
-- TOC entry 269 (class 1259 OID 33943)
-- Name: transaction_orders; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.transaction_orders (
    transaction_order_id uuid,
    transaction_id uuid,
    order_id character varying(100),
    organization_id uuid,
    created_at timestamp with time zone,
    created_by uuid,
    updated_at timestamp with time zone,
    updated_by uuid
);


ALTER TABLE public.transaction_orders OWNER TO postgres;

--
-- TOC entry 289 (class 1259 OID 35089)
-- Name: transaction_refund; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.transaction_refund (
    refund_id uuid,
    transaction_id uuid,
    reference_id character varying(50),
    description text,
    amount numeric,
    bank_code character varying(10),
    bank_account character varying(50),
    bank_account_name character varying(50),
    organization_id uuid,
    created_at timestamp with time zone,
    created_by uuid
);


ALTER TABLE public.transaction_refund OWNER TO postgres;

--
-- TOC entry 288 (class 1259 OID 35084)
-- Name: transaction_reimbursement; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.transaction_reimbursement (
    reimburse_id uuid,
    reference_id character varying(50),
    organization_id uuid,
    description text,
    amount numeric,
    status integer,
    employee_id uuid,
    payment_method integer,
    created_at timestamp with time zone,
    created_by uuid
);


ALTER TABLE public.transaction_reimbursement OWNER TO postgres;

--
-- TOC entry 267 (class 1259 OID 33935)
-- Name: transaction_types; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.transaction_types (
    type_id integer,
    type_label character varying
);


ALTER TABLE public.transaction_types OWNER TO postgres;

--
-- TOC entry 282 (class 1259 OID 34027)
-- Name: transactions; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.transactions (
    transaction_id uuid,
    transaction_type integer,
    reference_id character varying(50),
    order_type integer,
    transaction_category character varying(10),
    transaction_item character varying(10),
    invoice_number character varying,
    description text,
    transaction_date date,
    payment_type integer,
    organization_id uuid,
    amount numeric,
    bank_code character varying(10),
    bank_account character varying(20),
    payment_method integer,
    transaction_label character varying(50),
    note text,
    status integer,
    created_at timestamp with time zone,
    created_by uuid,
    updated_at timestamp with time zone,
    updated_by uuid
);


ALTER TABLE public.transactions OWNER TO postgres;

--
-- TOC entry 285 (class 1259 OID 34987)
-- Name: transacton_fleet_trips; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.transacton_fleet_trips (
    transaction_trip_id uuid,
    transaction_id uuid,
    schedule_number character varying(50),
    reference_id character varying(50),
    transaction_type integer,
    transaction_category character varying(10),
    transaction_item character varying(10),
    amount numeric,
    payment_type integer,
    description text,
    created_at timestamp with time zone,
    created_by uuid,
    updated_at timestamp with time zone,
    updated_by uuid,
    organization_id uuid
);


ALTER TABLE public.transacton_fleet_trips OWNER TO postgres;

--
-- TOC entry 305 (class 1259 OID 51563)
-- Name: travego_messages; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.travego_messages (
    message_id uuid,
    topic_id integer,
    fullname character varying(50),
    company_name character varying(50),
    email character varying(50),
    whatsapp character varying(20),
    scale character varying(10),
    messages text,
    is_read boolean,
    created_at timestamp with time zone
);


ALTER TABLE public.travego_messages OWNER TO postgres;

--
-- TOC entry 304 (class 1259 OID 51558)
-- Name: travego_reviews; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.travego_reviews (
    review_id uuid,
    user_id uuid,
    stars integer,
    review text,
    created_at timestamp with time zone,
    created_by uuid
);


ALTER TABLE public.travego_reviews OWNER TO postgres;

--
-- TOC entry 306 (class 1259 OID 51568)
-- Name: travego_transactions; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.travego_transactions (
    transaction_id uuid,
    transaction_date timestamp with time zone,
    invoice_number character varying(30),
    package_id character varying(10),
    start_date timestamp with time zone,
    expiry_date timestamp with time zone,
    payment_method character varying(20),
    payment_amount numeric,
    status integer,
    user_id uuid,
    organization_id uuid,
    created_at timestamp with time zone,
    created_by uuid,
    updated_at timestamp with time zone,
    updated_by uuid
);


ALTER TABLE public.travego_transactions OWNER TO postgres;

--
-- TOC entry 308 (class 1259 OID 51576)
-- Name: travego_visitors; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.travego_visitors (
    id integer NOT NULL,
    period date NOT NULL,
    count integer DEFAULT 0
);


ALTER TABLE public.travego_visitors OWNER TO postgres;

--
-- TOC entry 307 (class 1259 OID 51575)
-- Name: travego_visitors_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.travego_visitors_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.travego_visitors_id_seq OWNER TO postgres;

--
-- TOC entry 5315 (class 0 OID 0)
-- Dependencies: 307
-- Name: travego_visitors_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.travego_visitors_id_seq OWNED BY public.travego_visitors.id;


--
-- TOC entry 219 (class 1259 OID 17299)
-- Name: users; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.users (
    user_id uuid NOT NULL,
    username character varying(50),
    fullname character varying(100),
    email character varying(50),
    password text,
    phone character varying(20),
    address character varying(100),
    city character varying(30),
    province character varying(30),
    postal_code character varying(10),
    npwp character varying(25),
    date_of_birth timestamp with time zone,
    gender character varying(2),
    avatar character varying(255),
    is_active boolean,
    is_verified boolean,
    is_admin boolean,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    verified_at timestamp with time zone,
    last_login timestamp with time zone,
    deleted_at timestamp with time zone
);


ALTER TABLE public.users OWNER TO postgres;

--
-- TOC entry 5050 (class 2604 OID 51579)
-- Name: travego_visitors id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.travego_visitors ALTER COLUMN id SET DEFAULT nextval('public.travego_visitors_id_seq'::regclass);


--
-- TOC entry 5275 (class 0 OID 33985)
-- Dependencies: 275
-- Data for Name: _assistant; Type: TABLE DATA; Schema: public; Owner: postgres
--

--
-- TOC entry 5274 (class 0 OID 33980)
-- Dependencies: 274
-- Data for Name: _subscription_payment; Type: TABLE DATA; Schema: public; Owner: postgres
--

--
-- TOC entry 5235 (class 0 OID 25581)
-- Dependencies: 235
-- Data for Name: bank_list; Type: TABLE DATA; Schema: public; Owner: postgres
--

INSERT INTO public.bank_list VALUES ('011', 'BANK DANAMON INDONESIA', NULL);
INSERT INTO public.bank_list VALUES ('111', 'BANK DKI', NULL);
INSERT INTO public.bank_list VALUES ('046', 'BANK DBS INDONESIA', NULL);
INSERT INTO public.bank_list VALUES ('087', 'BANK HSBC INDONESIA', NULL);
INSERT INTO public.bank_list VALUES ('016', 'BANK MAYBANK INDONESIA, TBK', NULL);
INSERT INTO public.bank_list VALUES ('553', 'BANK MAYORA', NULL);
INSERT INTO public.bank_list VALUES ('426', 'BANK MEGA, TBK', NULL);
INSERT INTO public.bank_list VALUES ('147', 'BANK MUAMALAT INDONESIA, TBK', NULL);
INSERT INTO public.bank_list VALUES ('013', 'BANK PERMATA, TBK', NULL);
INSERT INTO public.bank_list VALUES ('721', 'BANK PERMATA, TBK UNIT USAHA SYARIAH', NULL);
INSERT INTO public.bank_list VALUES ('494', 'BANK RAKYAT INDONESIA AGRONIAGA, TBK', NULL);
INSERT INTO public.bank_list VALUES ('213', 'BANK TABUNGAN PENSIUNAN NASIONAL - (BTPN)', NULL);
INSERT INTO public.bank_list VALUES ('547', 'BANK TABUNGAN PENSIUNAN NASIONAL SYARIAH - (BTPN Syariah)', NULL);
INSERT INTO public.bank_list VALUES ('164', 'BANK ICBC INDONESIA', NULL);
INSERT INTO public.bank_list VALUES ('022', 'BANK CIMB NIAGA - (CIMB)', '/assets/bank-icon/cimb.png');
INSERT INTO public.bank_list VALUES ('730', 'BANK CIMB NIAGA UNIT USAHA SYARIAH - (CIMB SYARIAH)', '/assets/bank-icon/cimb.png');
INSERT INTO public.bank_list VALUES ('536', 'BANK BCA SYARIAH', '/assets/bank-icon/bca.png');
INSERT INTO public.bank_list VALUES ('014', 'BANK CENTRAL ASIA, TBK - (BCA)', '/assets/bank-icon/bca.png');
INSERT INTO public.bank_list VALUES ('427', 'BNI SYARIAH', '/assets/bank-icon/bni.png');
INSERT INTO public.bank_list VALUES ('009', 'BANK NEGARA INDONESIA (PERSERO), TBK (BNI)', '/assets/bank-icon/bni.png');
INSERT INTO public.bank_list VALUES ('008', 'BANK MANDIRI (PERSERO), TBK', '/assets/bank-icon/mandiri.png');
INSERT INTO public.bank_list VALUES ('564', 'BANK MANDIRI TASPEN POS', '/assets/bank-icon/mandiri.png');
INSERT INTO public.bank_list VALUES ('451', 'BANK SYARIAH MANDIRI', '/assets/bank-icon/mandiri.png');
INSERT INTO public.bank_list VALUES ('002', 'BANK RAKYAT INDONESIA (PERSERO), TBK (BRI)', '/assets/bank-icon/bri.png');
INSERT INTO public.bank_list VALUES ('422', 'BANK SYARIAH BRI - (BRI SYARIAH)', '/assets/bank-icon/bri.png');
INSERT INTO public.bank_list VALUES ('200', 'BANK TABUNGAN NEGARA (PERSERO), TBK (BTN)', '/assets/bank-icon/btn.png');
INSERT INTO public.bank_list VALUES ('723', 'BANK TABUNGAN NEGARA (PERSERO) SYARIAH (BTN Syariah)', '/assets/bank-icon/btn.png');
INSERT INTO public.bank_list VALUES ('028', 'BANK OCBC NISP, TBK', '/assets/bank-icon/ocbc.png');
INSERT INTO public.bank_list VALUES ('731', 'BANK OCBC NISP, TBK UNIT USAHA SYARIAH', '/assets/bank-icon/ocbc.png');
INSERT INTO public.bank_list VALUES ('441', 'BANK BUKOPIN', '/assets/bank-icon/bukopin.png');
INSERT INTO public.bank_list VALUES ('521', 'BANK SYARIAH BUKOPIN', '/assets/bank-icon/bukopin.png');

--
-- TOC entry 5266 (class 0 OID 33927)
-- Dependencies: 266
-- Data for Name: employee_leave_type; Type: TABLE DATA; Schema: public; Owner: postgres
--

INSERT INTO public.employee_leave_type VALUES (1, 'Cuti Reguler');
INSERT INTO public.employee_leave_type VALUES (2, 'Cuti / Izin Sakit');
INSERT INTO public.employee_leave_type VALUES (3, 'Izin Keluarga Sakit');
INSERT INTO public.employee_leave_type VALUES (4, 'Izin berduka');

--
-- TOC entry 5299 (class 0 OID 51513)
-- Dependencies: 299
-- Data for Name: facilities; Type: TABLE DATA; Schema: public; Owner: postgres
--

INSERT INTO public.facilities VALUES ('95376cd5-23c8-4d12-aeb1-d945aeedc70c', 'Pengemudi dan Pramusapa profesional', 'Smile               ', NULL);
INSERT INTO public.facilities VALUES ('93f60d8d-1c77-42eb-88e0-74174b7d2118', 'Power Plug Onboard', 'Cable               ', NULL);
INSERT INTO public.facilities VALUES ('ae22ca4e-1c03-4a09-a08a-954e876418f6', 'Support USB Cable', 'Usb                 ', NULL);
INSERT INTO public.facilities VALUES ('b757cb1c-2be5-4c0d-b013-102bc9c2021a', 'Termasuk Bahan Bakar', 'Fuel                ', NULL);
INSERT INTO public.facilities VALUES ('4ec69470-3ce7-479e-ac2d-5ff03e22b304', 'Pendingin Ruangan (AC)', 'Snowflake           ', NULL);
INSERT INTO public.facilities VALUES ('494fe96f-f505-4b89-aa80-56ca7b806faf', 'Air Suspension', 'RockingChair        ', NULL);
INSERT INTO public.facilities VALUES ('6c4e2ce1-a582-45c2-b7ed-04d81cb3b2b5', 'Recleaning Seat', 'Armchair            ', NULL);
INSERT INTO public.facilities VALUES ('3625ec50-2c12-4583-9223-2933c9096cf7', 'Movies & Entertaint', 'Clapperboard        ', NULL);
INSERT INTO public.facilities VALUES ('23324a46-1c62-4ef7-887f-612f469b1720', 'Alat Pemadam Api Ringan)', 'FireExtinguisher    ', NULL);
INSERT INTO public.facilities VALUES ('a892dc03-644d-45ed-89b1-653cc2470956', 'Snack & Makanan Ringan', 'Utensils            ', NULL);
INSERT INTO public.facilities VALUES ('b3d43245-3811-4fbd-980d-8d6268866328', 'Minuman Ringan', 'GlassWater          ', NULL);
INSERT INTO public.facilities VALUES ('14d25ab3-ed1a-4079-9a08-c01fcfd11921', 'Dilindungi Asuransi', 'ShieldCheck         ', NULL);
INSERT INTO public.facilities VALUES ('5b391476-cd97-4e20-8cf9-6f92dc39b518', 'Audio Video On Demand (AVOD)', 'Tv                  ', NULL);
INSERT INTO public.facilities VALUES ('16fad6ac-9b5e-4dd1-9182-8b5df31d8e0a', 'Tempat Sampah', 'Trash               ', NULL);
INSERT INTO public.facilities VALUES ('54483314-a961-4d92-a04a-471f1b3b1884', 'Cooling Box', 'Snowflake           ', NULL);
INSERT INTO public.facilities VALUES ('c6b4bd20-c41e-4e30-b7cc-8d4a0e2f5afd', 'Minibar and Dispenser', 'Wine                ', NULL);
INSERT INTO public.facilities VALUES ('d50898a8-fa04-48ca-a725-9ceb4408b71d', 'Toilet', 'Toilet              ', NULL);
INSERT INTO public.facilities VALUES ('bb98e836-508e-4941-bdd6-e361d80bafb9', 'Music and Karaoke', 'Music               ', NULL);

--
-- TOC entry 5239 (class 0 OID 25612)
-- Dependencies: 239
-- Data for Name: fleet_prices_history; Type: TABLE DATA; Schema: public; Owner: postgres
--



--
-- TOC entry 5225 (class 0 OID 17362)
-- Dependencies: 225
-- Data for Name: fleet_types; Type: TABLE DATA; Schema: public; Owner: postgres
--

INSERT INTO public.fleet_types VALUES ('FT01', 'Minibus');
INSERT INTO public.fleet_types VALUES ('FT03', 'Sedan');
INSERT INTO public.fleet_types VALUES ('FT04', 'MPV');
INSERT INTO public.fleet_types VALUES ('FT05', 'SUV');
INSERT INTO public.fleet_types VALUES ('FT06', 'Medium Bus');
INSERT INTO public.fleet_types VALUES ('FT07', 'Big Bus');
INSERT INTO public.fleet_types VALUES ('FT08', 'Double Decker');

--
-- TOC entry 5295 (class 0 OID 43310)
-- Dependencies: 295
-- Data for Name: inventory_movement_types; Type: TABLE DATA; Schema: public; Owner: postgres
--

INSERT INTO public.inventory_movement_types VALUES (1, 'Item Masuk');
INSERT INTO public.inventory_movement_types VALUES (2, 'Item Keluar');
INSERT INTO public.inventory_movement_types VALUES (3, 'Koreksi stok');
INSERT INTO public.inventory_movement_types VALUES (4, 'Transfer Stok');


--
-- TOC entry 5254 (class 0 OID 33879)
-- Dependencies: 254
-- Data for Name: organization_divisions; Type: TABLE DATA; Schema: public; Owner: postgres
--

INSERT INTO public.organization_divisions VALUES ('7c2a2d70-b542-4607-ba2b-d2087618e3a2', 'Marketing', 'Bertanggung jawab atas strategi pemasaran dan peningkatan volume penjualan.', '00000000-0000-0000-0000-000000000000', '2026-04-15 11:15:33.468247+07', '0cf12050-4ce1-44ac-855e-44110aecb6f6', NULL, NULL, 1);
INSERT INTO public.organization_divisions VALUES ('4df1996f-dd57-4586-a819-c2fe08107cf4', 'Finance', 'Mengelola administrasi keuangan, arus kas, serta pelaporan akuntansi', '00000000-0000-0000-0000-000000000000', '2026-04-15 11:30:48.521298+07', '0cf12050-4ce1-44ac-855e-44110aecb6f6', NULL, NULL, 1);
INSERT INTO public.organization_divisions VALUES ('fe8b3916-5eff-420c-8110-8d974d767afe', 'Operations', 'Mengoordinasikan pelaksanaan teknis perjalanan dan pemeliharaan armada operasional', '00000000-0000-0000-0000-000000000000', '2026-04-15 11:31:23.28055+07', '0cf12050-4ce1-44ac-855e-44110aecb6f6', '2026-04-15 16:02:18.752681+07', '0cf12050-4ce1-44ac-855e-44110aecb6f6', 1);


--
-- TOC entry 5238 (class 0 OID 25609)
-- Dependencies: 238
-- Data for Name: organization_members; Type: TABLE DATA; Schema: public; Owner: postgres
--



--
-- TOC entry 5253 (class 0 OID 33876)
-- Dependencies: 253
-- Data for Name: organization_roles; Type: TABLE DATA; Schema: public; Owner: postgres
--

INSERT INTO public.organization_roles VALUES ('0dbdb8c5-8edb-40ef-b0e3-3fd3d37daaa8', 'Pengemudi bertanggung jawab atas keselamatan penumpang dan pengoperasian armada kendaraan', 'Driver - Pengemudi', '00000000-0000-0000-0000-000000000000', '2026-04-15 16:21:21.153106+07', '0cf12050-4ce1-44ac-855e-44110aecb6f6', '2026-04-15 16:21:21.153106+07', '0cf12050-4ce1-44ac-855e-44110aecb6f6', 'fe8b3916-5eff-420c-8110-8d974d767afe', 1);
INSERT INTO public.organization_roles VALUES ('94acb1ae-07fa-44d7-b970-16b61d8aed25', 'Melakukan pemeliharaan rutin dan perbaikan teknis guna menjamin kelaikan armada', 'Mekanik', '00000000-0000-0000-0000-000000000000', '2026-04-15 16:22:35.00341+07', '0cf12050-4ce1-44ac-855e-44110aecb6f6', '2026-04-15 19:23:23.796214+07', '0cf12050-4ce1-44ac-855e-44110aecb6f6', 'fe8b3916-5eff-420c-8110-8d974d767afe', 1);
INSERT INTO public.organization_roles VALUES ('dd94c9a7-15fe-49c2-9c76-6e6472be67ec', 'Pemandu perjalanan pariwisata', 'Tour Guide', '00000000-0000-0000-0000-000000000000', '2026-04-15 19:23:42.300177+07', '0cf12050-4ce1-44ac-855e-44110aecb6f6', '2026-04-15 19:23:42.300177+07', '0cf12050-4ce1-44ac-855e-44110aecb6f6', 'fe8b3916-5eff-420c-8110-8d974d767afe', 1);


--
-- TOC entry 5220 (class 0 OID 17312)
-- Dependencies: 220
-- Data for Name: organization_types; Type: TABLE DATA; Schema: public; Owner: postgres
--

INSERT INTO public.organization_types VALUES (1, 'Travel Partner');
INSERT INTO public.organization_types VALUES (2, 'Biro Perjalanan dan Wisata');
INSERT INTO public.organization_types VALUES (3, 'Perusahaan Otobus');
INSERT INTO public.organization_types VALUES (4, 'Rental Armada Pariwisata');
INSERT INTO public.organization_types VALUES (5, 'Alat Berat');
INSERT INTO public.organization_types VALUES (6, 'Angkutan Ekspedisi dan Logistik');

--
-- TOC entry 5316 (class 0 OID 0)
-- Dependencies: 307
-- Name: travego_visitors_id_seq; Type: SEQUENCE SET; Schema: public; Owner: postgres
--

SELECT pg_catalog.setval('public.travego_visitors_id_seq', 26, true);


--
-- TOC entry 5062 (class 2606 OID 25587)
-- Name: bank_list bank_list_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.bank_list
    ADD CONSTRAINT bank_list_pkey PRIMARY KEY (code);


--
-- TOC entry 5066 (class 2606 OID 51584)
-- Name: travego_visitors travego_visitors_period_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.travego_visitors
    ADD CONSTRAINT travego_visitors_period_key UNIQUE (period);


--
-- TOC entry 5068 (class 2606 OID 51582)
-- Name: travego_visitors travego_visitors_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.travego_visitors
    ADD CONSTRAINT travego_visitors_pkey PRIMARY KEY (id);


--
-- TOC entry 5070 (class 2606 OID 51600)
-- Name: assistant_customer_stats unique_custstat_period_org_type_status; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.assistant_customer_stats
    ADD CONSTRAINT unique_custstat_period_org_type_status UNIQUE (period, type, status, organization_id);


--
-- TOC entry 5064 (class 2606 OID 51588)
-- Name: facilities unique_facility_per_org; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.facilities
    ADD CONSTRAINT unique_facility_per_org UNIQUE (organization_id, facility_name);


--
-- TOC entry 5072 (class 2606 OID 51598)
-- Name: assistant_account_stats unique_stat_period_org_type_status; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.assistant_account_stats
    ADD CONSTRAINT unique_stat_period_org_type_status UNIQUE (period, type, status, organization_id);


--
-- TOC entry 5060 (class 2606 OID 17305)
-- Name: users users_pkey1; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey1 PRIMARY KEY (user_id);


--
-- TOC entry 5058 (class 1259 OID 17306)
-- Name: idx_email_users; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_email_users ON public.users USING btree (email);


--
-- TOC entry 5054 (class 1259 OID 17274)
-- Name: idx_organization_users_created_by; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_organization_users_created_by ON public.organization_users USING btree (created_by);


--
-- TOC entry 5055 (class 1259 OID 17273)
-- Name: idx_organization_users_organization_id; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_organization_users_organization_id ON public.organization_users USING btree (organization_id);


--
-- TOC entry 5056 (class 1259 OID 17275)
-- Name: idx_organization_users_updated_by; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_organization_users_updated_by ON public.organization_users USING btree (updated_by);


--
-- TOC entry 5057 (class 1259 OID 17272)
-- Name: idx_organization_users_user_id; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_organization_users_user_id ON public.organization_users USING btree (user_id);


--
-- TOC entry 5052 (class 1259 OID 17255)
-- Name: idx_organizations_code; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_organizations_code ON public.organizations USING btree (organization_code);


--
-- TOC entry 5053 (class 1259 OID 17256)
-- Name: idx_organizations_created_by; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_organizations_created_by ON public.organizations USING btree (created_by);


--
-- TOC entry 5073 (class 2606 OID 17338)
-- Name: organizations organizations_created_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.organizations
    ADD CONSTRAINT organizations_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.users(user_id);


-- Completed on 2026-06-29 00:49:33

--
-- PostgreSQL database dump complete
--

