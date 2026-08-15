-- optional logo image for groups
ALTER TABLE groups
    ADD COLUMN logo_image BYTEA,
    ADD COLUMN logo_content_type VARCHAR(100);