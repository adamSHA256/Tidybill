package model

type PDFTemplate struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	TemplateCode string `json:"template_code"`
	ConfigJSON   string `json:"config_json"`
	IsDefault    bool   `json:"is_default"`
	SupplierID   string `json:"supplier_id"`
	Description  string `json:"description"`
	ShowLogo     bool   `json:"show_logo"`
	ShowQR       bool   `json:"show_qr"`
	ShowNotes    bool   `json:"show_notes"`
	PreviewPath  string `json:"preview_path"`
	SortOrder    int    `json:"sort_order"`
	IsBuiltin    bool   `json:"is_builtin"`
	YAMLSource   string `json:"yaml_source"`
	ParentID     string `json:"parent_id"`
}
