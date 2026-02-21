use std::fs::OpenOptions;
use std::io::Write;
use std::path::PathBuf;
use std::sync::{Arc, Mutex};
use tauri::Manager;
use tauri_plugin_shell::ShellExt;
use tauri_plugin_shell::process::{CommandChild, CommandEvent};

struct ApiPort(Arc<Mutex<Option<u16>>>);
struct SidecarChild(Mutex<Option<CommandChild>>);

/// Debug log file: %APPDATA%\TidyBill\debug.log (Win) or ~/.local/share/TidyBill/debug.log (Linux)
fn debug_log_path() -> Option<PathBuf> {
    std::env::var("APPDATA")
        .map(|p| PathBuf::from(p).join("TidyBill").join("debug.log"))
        .or_else(|_| {
            std::env::var("HOME")
                .map(|h| PathBuf::from(h).join(".local/share/TidyBill/debug.log"))
        })
        .ok()
}

fn dlog(msg: impl std::fmt::Display) {
    // Also print for dev console (works on Linux, no-op on Windows GUI)
    println!("[tidybill] {}", msg);
    if let Some(path) = debug_log_path() {
        if let Some(parent) = path.parent() {
            let _ = std::fs::create_dir_all(parent);
        }
        if let Ok(mut f) = OpenOptions::new().create(true).append(true).open(&path) {
            let _ = writeln!(f, "[tidybill] {}", msg);
        }
    }
}

#[tauri::command]
fn get_api_port(state: tauri::State<'_, ApiPort>) -> u16 {
    state.0.lock().unwrap().unwrap_or(0)
}

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    dlog("=== TidyBill desktop starting ===");

    let port_state = Arc::new(Mutex::new(None::<u16>));

    tauri::Builder::default()
        .plugin(tauri_plugin_shell::init())
        .plugin(tauri_plugin_dialog::init())
        .manage(ApiPort(port_state.clone()))
        .manage(SidecarChild(Mutex::new(None)))
        .invoke_handler(tauri::generate_handler![get_api_port])
        .setup(move |app| {
            let app_handle = app.handle().clone();
            let port_state = port_state.clone();

            let pid = std::process::id();
            dlog(format!("Tauri PID: {}", pid));
            dlog("Creating sidecar command...");

            // Spawn the Go sidecar backend with --gui --port 0 --parent-pid <pid>
            let sidecar_command = app_handle
                .shell()
                .sidecar("tidybill")
                .expect("failed to create sidecar command");

            dlog("Spawning sidecar...");
            let (mut rx, child) = sidecar_command
                .args(["--gui", "--port", "0", "--parent-pid", &pid.to_string()])
                .spawn()
                .expect("failed to spawn sidecar");

            dlog(format!("Sidecar spawned, child PID: {}", child.pid()));

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
                                    dlog(format!("Port captured: {}", port));
                                }
                            }
                            dlog(format!("stdout: {}", line));
                        }
                        CommandEvent::Stderr(line_bytes) => {
                            let line = String::from_utf8_lossy(&line_bytes);
                            dlog(format!("stderr: {}", line));
                        }
                        CommandEvent::Terminated(status) => {
                            dlog(format!("sidecar terminated: {:?}", status));
                        }
                        CommandEvent::Error(err) => {
                            dlog(format!("sidecar error: {}", err));
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
                dlog("App exit event — killing sidecar");
                if let Some(child) = app_handle.state::<SidecarChild>().0.lock().unwrap().take() {
                    let _ = child.kill();
                }
            }
        });
}
