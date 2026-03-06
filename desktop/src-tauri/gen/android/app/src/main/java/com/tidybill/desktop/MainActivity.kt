package com.tidybill.desktop

import android.os.Bundle
import android.util.Log
import mobile.Mobile

class MainActivity : TauriActivity() {
  companion object {
    private const val TAG = "TidyBill"
  }

  override fun onCreate(savedInstanceState: Bundle?) {
    super.onCreate(savedInstanceState)

    try {
      val port = Mobile.startServer(filesDir.absolutePath)
      Log.i(TAG, "Go backend started on port $port")
    } catch (e: Exception) {
      Log.e(TAG, "Failed to start Go backend", e)
    }
  }

  override fun onDestroy() {
    try {
      Mobile.stopServer()
      Log.i(TAG, "Go backend stopped")
    } catch (e: Exception) {
      Log.e(TAG, "Failed to stop Go backend", e)
    }
    super.onDestroy()
  }
}
