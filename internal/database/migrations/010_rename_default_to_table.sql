-- Rename template code 'default' to 'table'
UPDATE pdf_templates SET template_code = 'table' WHERE template_code = 'default';
UPDATE pdf_templates SET id = 'table' WHERE id = 'default';
UPDATE invoices SET template_id = 'table' WHERE template_id = 'default';
