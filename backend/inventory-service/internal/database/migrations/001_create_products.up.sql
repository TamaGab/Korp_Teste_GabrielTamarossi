CREATE TABLE IF NOT EXISTS products (
    id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    code VARCHAR(5) NOT NULL UNIQUE,
    description VARCHAR(200) NOT NULL,
    stock INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT products_code_format CHECK (code ~ '^[A-Z]{3}[0-9]{2}$'),
    CONSTRAINT products_description_not_blank CHECK (description ~ '[^[:space:]]'),
    CONSTRAINT products_stock_non_negative CHECK (stock >= 0)
);
