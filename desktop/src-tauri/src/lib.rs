use std::sync::{Arc, Mutex};
use tauri::Manager;
use tauri_plugin_shell::ShellExt;
use tauri_plugin_shell::process::{CommandChild, CommandEvent};

struct ApiPort(Arc<Mutex<Option<u16>>>);
struct SidecarChild(Mutex<Option<CommandChild>>);

#[tauri::command]
fn get_api_port(state: tauri::State<'_, ApiPort>) -> u16 {
    state.0.lock().unwrap().unwrap_or(0)
}

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    let port_state = Arc::new(Mutex::new(None::<u16>));

    tauri::Builder::default()
        .plugin(tauri_plugin_shell::init())
        .manage(ApiPort(port_state.clone()))
        .manage(SidecarChild(Mutex::new(None)))
        .invoke_handler(tauri::generate_handler![get_api_port])
        .setup(move |app| {
            let app_handle = app.handle().clone();
            let port_state = port_state.clone();

            let pid = std::process::id();

            // Spawn the Go sidecar backend with --gui --port 0 --parent-pid <pid>
            let sidecar_command = app_handle
                .shell()
                .sidecar("tidybill")
                .expect("failed to create sidecar command");

            let (mut rx, child) = sidecar_command
                .args(["--gui", "--port", "0", "--parent-pid", &pid.to_string()])
                .spawn()
                .expect("failed to spawn sidecar");

            // Store child handle so we can kill it on exit
            app_handle.state::<SidecarChild>().0.lock().unwrap().replace(child);

            // Listen for sidecar stdout/stderr in a background task
            tauri::async_runtime::spawn(async move {
                while let Some(event) = rx.recv().await {
                    match event {
                        CommandEvent::Stdout(line_bytes) => {
                            let line = String::from_utf8_lossy(&line_bytes);
                            if let Some(port_str) = line.strip_prefix("TIDYBILL_PORT=") {
                                if let Ok(port) = port_str.trim().parse::<u16>() {
                                    *port_state.lock().unwrap() = Some(port);
                                    println!("[tidybill] API port: {}", port);
                                }
                            }
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
        .build(tauri::generate_context!())
        .expect("error while building tauri application")
        .run(|app_handle, event| {
            if let tauri::RunEvent::Exit = event {
                if let Some(child) = app_handle.state::<SidecarChild>().0.lock().unwrap().take() {
                    println!("[tidybill] killing sidecar on exit");
                    let _ = child.kill();
                }
            }
        });
}
