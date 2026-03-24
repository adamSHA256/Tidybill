#[cfg(mobile)]
use tauri::Manager;
use tauri::{
    plugin::{Builder, TauriPlugin},
    Runtime,
};

#[cfg(desktop)]
#[allow(dead_code)]
mod desktop;
#[cfg(mobile)]
mod mobile;

mod error;

pub use error::{Error, Result};

#[cfg(mobile)]
#[tauri::command]
async fn share_file<R: Runtime>(
    app: tauri::AppHandle<R>,
    file_path: String,
    mime_type: String,
    title: Option<String>,
) -> std::result::Result<(), String> {
    use mobile::Sharesheet;
    app.state::<Sharesheet<R>>()
        .share_file(file_path, mime_type, title)
        .map_err(|e| e.to_string())
}

#[cfg(desktop)]
#[tauri::command]
async fn share_file(
    _file_path: String,
    _mime_type: String,
    _title: Option<String>,
) -> std::result::Result<(), String> {
    // No-op on desktop
    Ok(())
}

pub fn init<R: Runtime>() -> TauriPlugin<R> {
    Builder::new("sharesheet")
        .setup(|app, api| {
            #[cfg(mobile)]
            {
                let handle = mobile::init(app, api)?;
                app.manage(handle);
            }
            #[cfg(desktop)]
            {
                let _ = (app, api);
            }
            Ok(())
        })
        .invoke_handler(tauri::generate_handler![share_file])
        .build()
}
