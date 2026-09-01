CREATE TABLE IF NOT EXISTS invoices (
    number BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    status TEXT NOT NULL DEFAULT 'OPEN',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT invoices_status_valid CHECK (status IN ('OPEN', 'CLOSED'))
);

CREATE TABLE IF NOT EXISTS invoice_lines (
    invoice_number BIGINT NOT NULL REFERENCES invoices(number),
    inventory_product_id INTEGER NOT NULL,
    product_code VARCHAR(5) NOT NULL,
    product_description VARCHAR(200) NOT NULL,
    quantity INTEGER NOT NULL,
    PRIMARY KEY (invoice_number, inventory_product_id),
    CONSTRAINT invoice_lines_inventory_product_id_positive CHECK (inventory_product_id > 0),
    CONSTRAINT invoice_lines_product_code_format CHECK (product_code ~ '^[A-Z]{3}[0-9]{2}$'),
    CONSTRAINT invoice_lines_description_not_blank CHECK (product_description ~ '[^[:space:]]'),
    CONSTRAINT invoice_lines_quantity_positive CHECK (quantity > 0)
);
