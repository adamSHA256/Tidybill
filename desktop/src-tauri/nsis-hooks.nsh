!macro NSIS_HOOK_POSTUNINSTALL
  ; Skip data deletion when updating (uninstall-before-reinstall flow)
  ${If} $UpdateMode <> 1
    MessageBox MB_YESNO "Delete all TidyBill data (database, settings, logos)?" /SD IDNO IDYES _delete_data IDNO _skip_data
    _delete_data:
      RMDir /r "$APPDATA\TidyBill"
    _skip_data:
  ${EndIf}
!macroend
