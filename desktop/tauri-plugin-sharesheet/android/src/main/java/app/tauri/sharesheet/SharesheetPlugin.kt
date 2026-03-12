package app.tauri.sharesheet

import android.app.Activity
import android.content.Intent
import android.util.Log
import androidx.core.content.FileProvider
import app.tauri.annotation.Command
import app.tauri.annotation.InvokeArg
import app.tauri.annotation.TauriPlugin
import app.tauri.plugin.Invoke
import app.tauri.plugin.JSObject
import app.tauri.plugin.Plugin
import java.io.File

@InvokeArg
class ShareFileArgs {
    lateinit var filePath: String
    var mimeType: String = "application/pdf"
    var title: String = ""
}

@TauriPlugin
class SharesheetPlugin(private val activity: Activity) : Plugin(activity) {

    companion object {
        private const val TAG = "SharesheetPlugin"
    }

    @Command
    fun shareFile(invoke: Invoke) {
        try {
            val args = invoke.parseArgs(ShareFileArgs::class.java)
            val file = File(args.filePath)

            if (!file.exists()) {
                invoke.reject("File not found: ${args.filePath}")
                return
            }

            val authority = "${activity.packageName}.fileprovider"
            val contentUri = FileProvider.getUriForFile(activity, authority, file)

            Log.d(TAG, "Sharing file: ${args.filePath}")
            Log.d(TAG, "Content URI: $contentUri")

            val sendIntent = Intent().apply {
                action = Intent.ACTION_SEND
                type = args.mimeType
                putExtra(Intent.EXTRA_STREAM, contentUri)
                if (args.title.isNotEmpty()) {
                    putExtra(Intent.EXTRA_TITLE, args.title)
                    putExtra(Intent.EXTRA_SUBJECT, args.title)
                }
                addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION)
            }

            val chooser = Intent.createChooser(sendIntent, args.title.ifEmpty { "Share" })
            chooser.addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
            activity.startActivity(chooser)

            invoke.resolve(JSObject())
        } catch (e: Exception) {
            Log.e(TAG, "Failed to share file", e)
            invoke.reject("Failed to share file: ${e.message}")
        }
    }
}
