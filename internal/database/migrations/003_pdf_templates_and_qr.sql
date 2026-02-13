-- PDF Templates expansion + QR type support

-- Expand pdf_templates table with config columns
ALTER TABLE pdf_templates ADD COLUMN description TEXT DEFAULT '';
ALTER TABLE pdf_templates ADD COLUMN show_logo INTEGER DEFAULT 1;
ALTER TABLE pdf_templates ADD COLUMN show_qr INTEGER DEFAULT 1;
ALTER TABLE pdf_templates ADD COLUMN show_notes INTEGER DEFAULT 1;
ALTER TABLE pdf_templates ADD COLUMN preview_path TEXT DEFAULT '';
ALTER TABLE pdf_templates ADD COLUMN sort_order INTEGER DEFAULT 0;

-- Add QR code type to bank_accounts
ALTER TABLE bank_accounts ADD COLUMN qr_type TEXT DEFAULT 'spayd';

-- Track which template was used for each invoice
ALTER TABLE invoices ADD COLUMN template_id TEXT DEFAULT 'default';

-- Seed the 3 generator templates + update existing default
UPDATE pdf_templates SET name = 'Výchozí', description = 'Původní šablona aplikace s šedým záhlavím a plnými okraji tabulky', sort_order = 0 WHERE id = 'default';

INSERT OR IGNORE INTO pdf_templates (id, name, template_code, description, is_default, show_logo, show_qr, show_notes, sort_order) VALUES
    ('classic', 'Klasická', 'classic', 'Tradiční černobílá faktura s čísly stránek a formálním rozložením', 0, 1, 1, 1, 1),
    ('modern',  'Moderní',  'modern',  'Současný design s ocelovým modrým akcentem a barevnou hierarchií', 0, 1, 1, 1, 2),
    ('minimal', 'Minimální','minimal', 'Minimalistický styl s maximem bílého prostoru a bez dekorací', 0, 0, 1, 0, 3);
