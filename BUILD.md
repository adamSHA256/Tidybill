## Releasing

For published releases to include a working "Connect to Google Drive" button, two GitHub Actions secrets must be configured on the repo:

- `GDRIVE_CLIENT_ID`
- `GDRIVE_CLIENT_SECRET`

Get these from the Google Cloud Console OAuth Desktop client at <https://console.cloud.google.com/apis/credentials>. Add via Settings → Secrets and variables → Actions → New repository secret.

If unset, the build succeeds and the GDrive Connect button becomes a runtime no-op with a clear error message — non-blocking but should be set before any release tag.
