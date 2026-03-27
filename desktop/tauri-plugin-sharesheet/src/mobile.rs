use serde::de::DeserializeOwned;
use tauri::{
    plugin::{PluginApi, PluginHandle},
    AppHandle, Runtime,
};

#[cfg(target_os = "android")]
const PLUGIN_IDENTIFIER: &str = "app.tauri.sharesheet";

pub fn init<R: Runtime, C: DeserializeOwned>(
    _app: &AppHandle<R>,
    api: PluginApi<R, C>,
) -> crate::Result<Sharesheet<R>> {
    #[cfg(target_os = "android")]
    let handle = api.register_android_plugin(PLUGIN_IDENTIFIER, "SharesheetPlugin")?;
    Ok(Sharesheet(handle))
}

pub struct Sharesheet<R: Runtime>(PluginHandle<R>);

impl<R: Runtime> Sharesheet<R> {
    pub fn share_file(
        &self,
        file_path: String,
        mime_type: String,
        title: Option<String>,
    ) -> crate::Result<()> {
        // The Kotlin plugin returns JSObject() which serializes as `{}`.
        // Rust cannot deserialize `{}` as `()` (unit), so we deserialize
        // into serde_json::Value and discard the result.
        self.0
            .run_mobile_plugin::<serde_json::Value>(
                "shareFile",
                serde_json::json!({
                    "filePath": file_path,
                    "mimeType": mime_type,
                    "title": title.unwrap_or_default(),
                }),
            )
            .map(|_| ())
            .map_err(Into::into)
    }
}
