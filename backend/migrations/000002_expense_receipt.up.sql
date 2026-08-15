-- optional receipt image for expenses
ALTER TABLE expenses
    ADD COLUMN receipt_image BYTEA,
    ADD COLUMN receipt_content_type VARCHAR(100);