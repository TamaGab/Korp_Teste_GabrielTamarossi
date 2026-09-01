ALTER TABLE invoice_lines DROP CONSTRAINT IF EXISTS invoice_lines_invoice_number_fkey;
ALTER TABLE invoice_lines
    ADD CONSTRAINT invoice_lines_invoice_number_fkey
    FOREIGN KEY (invoice_number) REFERENCES invoices(number) ON DELETE CASCADE;

ALTER TABLE invoice_lines RENAME COLUMN inventory_product_id TO product_id;
