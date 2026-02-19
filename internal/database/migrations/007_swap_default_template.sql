-- Make 'classic' the default template instead of 'default'
UPDATE pdf_templates SET is_default = 1 WHERE template_code = 'classic';
UPDATE pdf_templates SET is_default = 0 WHERE template_code = 'default';

-- Reorder: classic=0 (first), default=1, modern=2, minimal=3
UPDATE pdf_templates SET sort_order = 0 WHERE template_code = 'classic';
UPDATE pdf_templates SET sort_order = 1 WHERE template_code = 'default';
UPDATE pdf_templates SET sort_order = 2 WHERE template_code = 'modern';
UPDATE pdf_templates SET sort_order = 3 WHERE template_code = 'minimal';
