package rclone

type BackendField struct {
	Name    string   // rclone option name, e.g. "host"
	Label   string   // translation key, e.g. "cloud.rclone.sftp.host"
	Kind    string   // "text" | "password" | "number" | "select" | "checkbox"
	Required bool
	Default string
	Options []string // for Kind == "select"
	Obscure bool     // pass opt.obscure = true for this field's value? (rclone auto-obscures pass fields when opt.obscure = true globally)
}

type Backend struct {
	ID     string // "sftp", "webdav", "s3", "dropbox", "onedrive"
	Type   string // the rclone backend type; usually equals ID
	Fields []BackendField
}

var Backends = []Backend{
	{
		ID: "sftp", Type: "sftp",
		Fields: []BackendField{
			{Name: "host", Kind: "text", Required: true},
			{Name: "port", Kind: "number", Default: "22"},
			{Name: "user", Kind: "text", Required: true},
			{Name: "pass", Kind: "password", Obscure: true},
			{Name: "key_file", Kind: "text"},
			{Name: "key_file_pass", Kind: "password", Obscure: true},
		},
	},
	{
		ID: "webdav", Type: "webdav",
		Fields: []BackendField{
			{Name: "url", Kind: "text", Required: true},
			{Name: "vendor", Kind: "select", Default: "nextcloud",
				Options: []string{"nextcloud", "owncloud", "sharepoint", "other"}},
			{Name: "user", Kind: "text"},
			{Name: "pass", Kind: "password", Obscure: true},
		},
	},
	{
		ID: "s3", Type: "s3",
		Fields: []BackendField{
			{Name: "provider", Kind: "select", Required: true,
				Options: []string{"AWS", "Backblaze", "Cloudflare", "Minio", "Other"}},
			{Name: "access_key_id",     Kind: "text",     Required: true},
			{Name: "secret_access_key", Kind: "password", Required: true},
			{Name: "region",            Kind: "text",     Default: "us-east-1"},
			{Name: "endpoint",          Kind: "text"},
			// bucket is not a backend field — it is part of the path. The frontend collects it and we prepend it to the folder path.
		},
	},
	{
		ID: "dropbox", Type: "dropbox",
		// No fields — uses rclone's interactive auth. Not implemented in v1;
		// the wizard greys out Dropbox until we implement the OAuth-in-rclone flow.
		Fields: []BackendField{},
	},
	{
		ID: "onedrive", Type: "onedrive",
		Fields: []BackendField{},
	},
}

func FindBackend(id string) *Backend {
	for i := range Backends {
		if Backends[i].ID == id {
			return &Backends[i]
		}
	}
	return nil
}
