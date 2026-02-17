/**
 * Apply zoom level to the app.
 * Uses Tauri's native webview zoom when available, falls back to CSS zoom.
 */
export async function applyZoom(factor: number) {
  try {
    const { getCurrentWebview } = await import('@tauri-apps/api/webview')
    await getCurrentWebview().setZoom(factor)
  } catch {
    // Fallback for dev mode or non-Tauri environments
    document.documentElement.style.zoom = String(factor)
  }
}
