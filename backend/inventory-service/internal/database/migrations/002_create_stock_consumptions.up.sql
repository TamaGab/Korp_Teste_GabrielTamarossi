CREATE TABLE IF NOT EXISTS stock_consumptions (
    invoice_number TEXT PRIMARY KEY,
    payload_hash BYTEA NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT stock_consumptions_invoice_number_not_blank CHECK (invoice_number ~ '[^[:space:]]')
);
