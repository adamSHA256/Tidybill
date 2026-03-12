use serde::de::DeserializeOwned;
use tauri::{plugin::PluginApi, AppHandle, Runtime};

pub fn init<R: Runtime, C: DeserializeOwned>(
    _app: &AppHandle<R>,
    _api: PluginApi<R, C>,
) -> crate::Result<Sharesheet> {
    Ok(Sharesheet)
}

pub struct Sharesheet;

impl Sharesheet {
    pub fn share_file(
        &self,
        _file_path: String,
        _mime_type: String,
        _title: Option<String>,
    ) -> crate::Result<()> {
        // Sharing via native sheet is not available on desktop
        Ok(())
    }
}
