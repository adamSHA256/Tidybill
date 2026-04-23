package gdrive

// ClientID and ClientSecret are injected at build time via
// -ldflags "-X github.com/adamSHA256/tidybill/internal/cloud/gdrive.ClientID=... -X .../ClientSecret=..."
// For local development an unprivileged dev OAuth client may be used.
var (
	ClientID     = ""
	ClientSecret = ""
)

const (
	TransportID = "gdrive"

	ScopeDriveFile = "https://www.googleapis.com/auth/drive.file"
	ScopeOpenID    = "openid"
	ScopeEmail     = "email"

	AuthURL     = "https://accounts.google.com/o/oauth2/v2/auth"
	TokenURL    = "https://oauth2.googleapis.com/token"
	RevokeURL   = "https://oauth2.googleapis.com/revoke"
	UserInfoURL = "https://openidconnect.googleapis.com/v1/userinfo"

	FolderName = "TidyBill"
	FolderMime = "application/vnd.google-apps.folder"
)
