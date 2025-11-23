CREATE TABLE public.users
(
    user_id uuid NOT NULL,
    username character varying(50),
    fullname character varying(100),
    email character varying(50),
    password text,
    phone character varying(20),
    address character varying(100),
    avatar character varying(200),
    city character varying(30),
    province character varying(30),
    postal_code character varying(10),
    npwp character varying(25),
    date_of_birth timestamp with time zone,
    gender integer,
    is_active boolean,
    is_verified boolean,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    verified_at timestamp with time zone,
    last_login timestamp with time zone,
    PRIMARY KEY (user_id)
);

ALTER TABLE IF EXISTS public.users OWNER to postgres;

CREATE INDEX IF NOT EXISTS idx_users_email ON public.users(email);

