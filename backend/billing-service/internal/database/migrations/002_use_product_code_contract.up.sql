DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_name = 'invoice_lines' AND column_name = 'product_id'
    ) THEN
        ALTER TABLE invoice_lines RENAME COLUMN product_id TO inventory_product_id;
    END IF;
END $$;

ALTER TABLE invoice_lines DROP CONSTRAINT IF EXISTS invoice_lines_invoice_number_fkey;
ALTER TABLE invoice_lines
    ADD CONSTRAINT invoice_lines_invoice_number_fkey
    FOREIGN KEY (invoice_number) REFERENCES invoices(number);
