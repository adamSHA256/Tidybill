use tauri_plugin_shell::ShellExt;
use tauri_plugin_shell::process::CommandEvent;

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .plugin(tauri_plugin_shell::init())
        .setup(|app| {
            let app_handle = app.handle().clone();

            // Spawn the Go sidecar backend with --gui flag
            let sidecar_command = app_handle
                .shell()
                .sidecar("tidybill")
                .expect("failed to create sidecar command");

            let (mut rx, _child) = sidecar_command
                .args(["--gui"])
                .spawn()
                .expect("failed to spawn sidecar");

            // Listen for sidecar stdout/stderr in a background task
            tauri::async_runtime::spawn(async move {
                while let Some(event) = rx.recv().await {
                    match event {
                        CommandEvent::Stdout(line_bytes) => {
                            let line = String::from_utf8_lossy(&line_bytes);
                            println!("[tidybill] {}", line);
                        }
                        CommandEvent::Stderr(line_bytes) => {
                            let line = String::from_utf8_lossy(&line_bytes);
                            eprintln!("[tidybill] {}", line);
                        }
                        CommandEvent::Terminated(status) => {
                            println!("[tidybill] terminated with status: {:?}", status);
                        }
                        CommandEvent::Error(err) => {
                            eprintln!("[tidybill] error: {}", err);
                        }
                        _ => {}
                    }
                }
            });

            Ok(())
        })
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
