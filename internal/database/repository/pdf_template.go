package repository

import (
	"database/sql"

	"github.com/adamSHA256/tidybill/internal/model"
)

type PDFTemplateRepository struct {
	db DBTX
}

func NewPDFTemplateRepository(db DBTX) *PDFTemplateRepository {
	return &PDFTemplateRepository{db: db}
}

func (r *PDFTemplateRepository) List() ([]*model.PDFTemplate, error) {
	rows, err := r.db.Query(`
		SELECT id, name, template_code, COALESCE(config_json, ''), is_default, COALESCE(supplier_id, ''),
			description, show_logo, show_qr, show_notes, COALESCE(preview_path, ''), sort_order
		FROM pdf_templates ORDER BY sort_order ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []*model.PDFTemplate
	for rows.Next() {
		t := &model.PDFTemplate{}
		if err := rows.Scan(&t.ID, &t.Name, &t.TemplateCode, &t.ConfigJSON, &t.IsDefault,
			&t.SupplierID, &t.Description, &t.ShowLogo, &t.ShowQR, &t.ShowNotes,
			&t.PreviewPath, &t.SortOrder); err != nil {
			return nil, err
		}
		templates = append(templates, t)
	}
	return templates, rows.Err()
}

func (r *PDFTemplateRepository) GetByID(id string) (*model.PDFTemplate, error) {
	t := &model.PDFTemplate{}
	err := r.db.QueryRow(`
		SELECT id, name, template_code, COALESCE(config_json, ''), is_default, COALESCE(supplier_id, ''),
			description, show_logo, show_qr, show_notes, COALESCE(preview_path, ''), sort_order
		FROM pdf_templates WHERE id = ?`, id).Scan(
		&t.ID, &t.Name, &t.TemplateCode, &t.ConfigJSON, &t.IsDefault,
		&t.SupplierID, &t.Description, &t.ShowLogo, &t.ShowQR, &t.ShowNotes,
		&t.PreviewPath, &t.SortOrder)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return t, err
}

func (r *PDFTemplateRepository) GetDefault() (*model.PDFTemplate, error) {
	t := &model.PDFTemplate{}
	err := r.db.QueryRow(`
		SELECT id, name, template_code, COALESCE(config_json, ''), is_default, COALESCE(supplier_id, ''),
			description, show_logo, show_qr, show_notes, COALESCE(preview_path, ''), sort_order
		FROM pdf_templates WHERE is_default = 1 LIMIT 1`).Scan(
		&t.ID, &t.Name, &t.TemplateCode, &t.ConfigJSON, &t.IsDefault,
		&t.SupplierID, &t.Description, &t.ShowLogo, &t.ShowQR, &t.ShowNotes,
		&t.PreviewPath, &t.SortOrder)
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
	// Must be atomic: unset all defaults, then set the new one
	_, err := r.db.Exec(`UPDATE pdf_templates SET is_default = 0`)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(`UPDATE pdf_templates SET is_default = 1 WHERE id = ?`, id)
	return err
}
