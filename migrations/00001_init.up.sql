CREATE TABLE products
(
    sku          TEXT PRIMARY KEY,
    name         TEXT        NOT NULL,
    product_type TEXT        NOT NULL,
    price        BIGINT      NOT NULL,
    currency     TEXT        NOT NULL,
    image        TEXT        NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL
);

CREATE TABLE orders
(
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    status     TEXT        NOT NULL,
    sku        TEXT        NOT NULL REFERENCES products (sku),
    amount     BIGINT      NOT NULL,
    code       TEXT,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE keys
(
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    sku        TEXT        NOT NULL,
    value      TEXT UNIQUE NOT NULL,
    is_used    BOOLEAN     NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE issues
(
    request_id TEXT PRIMARY KEY,
    key_id     BIGINT      NOT NULL UNIQUE REFERENCES keys (id),
    created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE delivery_attempts
(
    id             BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    request_id     TEXT        NOT NULL,
    order_id       BIGINT      NOT NULL REFERENCES orders (id),
    provider       TEXT        NOT NULL,
    status         TEXT        NOT NULL,
    attempt_number BIGINT      NOT NULL,
    process_after  TIMESTAMPTZ NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL,
    updated_at     TIMESTAMPTZ NOT NULL
);

CREATE INDEX delivery_attempts_status_process_after_idx ON delivery_attempts (status, process_after);

CREATE TABLE webhooks
(
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    event_id   TEXT        NOT NULL UNIQUE,
    order_id   BIGINT      NOT NULL REFERENCES orders (id),
    status     TEXT        NOT NULL,
    amount     BIGINT      NOT NULL,
    currency   TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);



