package repository

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"

	"github.com/adamSHA256/tidybill/internal/model"
)

type PDFTemplateRepository struct {
	db DBTX
}

func NewPDFTemplateRepository(db DBTX) *PDFTemplateRepository {
	return &PDFTemplateRepository{db: db}
}

const templateColumns = `id, name, template_code, COALESCE(config_json, ''), is_default, COALESCE(supplier_id, ''),
	description, show_logo, show_qr, show_notes, COALESCE(preview_path, ''), sort_order,
	COALESCE(is_builtin, 0), COALESCE(yaml_source, ''), COALESCE(parent_id, '')`

func scanTemplate(row interface{ Scan(dest ...any) error }) (*model.PDFTemplate, error) {
	t := &model.PDFTemplate{}
	err := row.Scan(&t.ID, &t.Name, &t.TemplateCode, &t.ConfigJSON, &t.IsDefault,
		&t.SupplierID, &t.Description, &t.ShowLogo, &t.ShowQR, &t.ShowNotes,
		&t.PreviewPath, &t.SortOrder, &t.IsBuiltin, &t.YAMLSource, &t.ParentID)
	return t, err
}

func (r *PDFTemplateRepository) List() ([]*model.PDFTemplate, error) {
	rows, err := r.db.Query(`SELECT ` + templateColumns + ` FROM pdf_templates ORDER BY sort_order ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []*model.PDFTemplate
	for rows.Next() {
		t, err := scanTemplate(rows)
		if err != nil {
			return nil, err
		}
		templates = append(templates, t)
	}
	return templates, rows.Err()
}

func (r *PDFTemplateRepository) GetByID(id string) (*model.PDFTemplate, error) {
	row := r.db.QueryRow(`SELECT `+templateColumns+` FROM pdf_templates WHERE id = ?`, id)
	t, err := scanTemplate(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return t, err
}

func (r *PDFTemplateRepository) GetDefault() (*model.PDFTemplate, error) {
	row := r.db.QueryRow(`SELECT ` + templateColumns + ` FROM pdf_templates WHERE is_default = 1 LIMIT 1`)
	t, err := scanTemplate(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return t, err
}

func (r *PDFTemplateRepository) Update(t *model.PDFTemplate) error {
	_, err := r.db.Exec(`
		UPDATE pdf_templates SET name=?, description=?, show_logo=?, show_qr=?, show_notes=?, preview_path=?
		WHERE id=?`,
		t.Name, t.Description, t.ShowLogo, t.ShowQR, t.ShowNotes, t.PreviewPath, t.ID)
	return err
}

func (r *PDFTemplateRepository) SetDefault(id string) error {
	_, err := r.db.Exec(`UPDATE pdf_templates SET is_default = 0`)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(`UPDATE pdf_templates SET is_default = 1 WHERE id = ?`, id)
	return err
}

func (r *PDFTemplateRepository) Duplicate(sourceID, newName string) (*model.PDFTemplate, error) {
	src, err := r.GetByID(sourceID)
	if err != nil {
		return nil, err
	}
	if src == nil {
		return nil, fmt.Errorf("source template not found: %s", sourceID)
	}

	// Get max sort_order
	var maxSort int
	r.db.QueryRow(`SELECT COALESCE(MAX(sort_order), 0) FROM pdf_templates`).Scan(&maxSort)

	newID := uuid.New().String()[:8]
	templateCode := "custom_" + newID

	_, err = r.db.Exec(`
		INSERT INTO pdf_templates (id, name, template_code, is_default, description, show_logo, show_qr, show_notes, sort_order, is_builtin, yaml_source, parent_id)
		VALUES (?, ?, ?, 0, ?, ?, ?, ?, ?, 0, ?, ?)`,
		newID, newName, templateCode, src.Description, src.ShowLogo, src.ShowQR, src.ShowNotes,
		maxSort+1, src.YAMLSource, sourceID)
	if err != nil {
		return nil, err
	}

	return r.GetByID(newID)
}

func (r *PDFTemplateRepository) UpdateYAMLSource(id, yaml string) error {
	_, err := r.db.Exec(`UPDATE pdf_templates SET yaml_source = ? WHERE id = ? AND is_builtin = 0`, yaml, id)
	return err
}

func (r *PDFTemplateRepository) Delete(id string) error {
	// Prevent deleting built-in templates
	var isBuiltin bool
	err := r.db.QueryRow(`SELECT is_builtin FROM pdf_templates WHERE id = ?`, id).Scan(&isBuiltin)
	if err != nil {
		return err
	}
	if isBuiltin {
		return fmt.Errorf("cannot delete built-in template")
	}

	// Check if any invoices reference this template
	var count int
	err = r.db.QueryRow(`SELECT COUNT(*) FROM invoices WHERE template_id = ?`, id).Scan(&count)
	if err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("cannot delete template: %d invoices reference it", count)
	}

	_, err = r.db.Exec(`DELETE FROM pdf_templates WHERE id = ? AND is_builtin = 0`, id)
	return err
}
