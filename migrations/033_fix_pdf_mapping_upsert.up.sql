-- Fix pdf_product_mappings: add unique constraint on pdf_product_text
-- (no duplicates exist, safe to add directly)
ALTER TABLE pdf_product_mappings
    ADD CONSTRAINT uq_pdf_product_text UNIQUE (pdf_product_text);

-- Fix pdf_package_mappings: deduplicate and add unique constraint on pdf_package_text
-- Keep the row with highest usage_count (or lowest mapping_id as tiebreak)
DELETE FROM pdf_package_mappings
WHERE mapping_id NOT IN (
    SELECT DISTINCT ON (pdf_package_text) mapping_id
    FROM pdf_package_mappings
    ORDER BY pdf_package_text, usage_count DESC, mapping_id ASC
);

ALTER TABLE pdf_package_mappings
    ADD CONSTRAINT uq_pdf_package_text UNIQUE (pdf_package_text);
