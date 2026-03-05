use std::sync::{Arc, Mutex};

#[cfg(desktop)]
use tauri::Manager;
#[cfg(desktop)]
use std::fs::OpenOptions;
#[cfg(desktop)]
use std::io::Write;
#[cfg(desktop)]
use std::path::PathBuf;
#[cfg(desktop)]
use tauri_plugin_shell::ShellExt;
#[cfg(desktop)]
use tauri_plugin_shell::process::{CommandChild, CommandEvent};

struct ApiPort(Arc<Mutex<Option<u16>>>);

#[cfg(desktop)]
struct SidecarChild(Mutex<Option<CommandChild>>);

#[cfg(desktop)]
fn debug_log_path() -> Option<PathBuf> {
    std::env::var("APPDATA")
        .map(|p| PathBuf::from(p).join("TidyBill").join("debug.log"))
        .or_else(|_| {
            std::env::var("HOME")
                .map(|h| PathBuf::from(h).join(".local/share/TidyBill/debug.log"))
        })
        .ok()
}

#[cfg(desktop)]
fn dlog(msg: impl std::fmt::Display) {
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
    let port_state = Arc::new(Mutex::new(None::<u16>));

    let mut builder = tauri::Builder::default()
        .plugin(tauri_plugin_dialog::init())
        .manage(ApiPort(port_state.clone()))
        .invoke_handler(tauri::generate_handler![get_api_port]);

    #[cfg(desktop)]
    {
        dlog("=== TidyBill desktop starting ===");
        builder = builder
            .plugin(tauri_plugin_shell::init())
            .manage(SidecarChild(Mutex::new(None)))
            .setup(move |app| {
                let app_handle = app.handle().clone();
                let port_state = port_state.clone();

                let pid = std::process::id();
                dlog(format!("Tauri PID: {}", pid));
                dlog("Creating sidecar command...");

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
                app_handle.state::<SidecarChild>().0.lock().unwrap().replace(child);

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
            });
    }

    #[cfg(mobile)]
    {
        builder = builder.setup(move |_app| {
            // Go backend is started by the Kotlin Activity (via gomobile AAR).
            // It listens on a fixed port for the PoC.
            *port_state.lock().unwrap() = Some(18080);
            Ok(())
        });
    }

    builder
        .build(tauri::generate_context!())
        .expect("error while building tauri application")
        .run(|_app_handle, event| {
            #[cfg(desktop)]
            if let tauri::RunEvent::Exit = event {
                dlog("App exit event — killing sidecar");
                if let Some(child) = _app_handle.state::<SidecarChild>().0.lock().unwrap().take() {
                    let _ = child.kill();
                }
            }
            let _ = event; // suppress unused warning on mobile
        });
}
