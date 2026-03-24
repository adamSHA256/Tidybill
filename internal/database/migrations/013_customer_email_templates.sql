ALTER TABLE customers ADD COLUMN email_custom_template INTEGER DEFAULT 0;
ALTER TABLE customers ADD COLUMN email_subject_template TEXT DEFAULT '';
ALTER TABLE customers ADD COLUMN email_body_template TEXT DEFAULT '';
