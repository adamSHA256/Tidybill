-- Custom templates support: allow users to duplicate and edit templates
ALTER TABLE pdf_templates ADD COLUMN is_builtin INTEGER DEFAULT 0;
ALTER TABLE pdf_templates ADD COLUMN yaml_source TEXT DEFAULT '';
ALTER TABLE pdf_templates ADD COLUMN parent_id TEXT DEFAULT '';

-- Mark existing 4 templates as built-in
UPDATE pdf_templates SET is_builtin = 1;
