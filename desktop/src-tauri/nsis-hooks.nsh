!macro NSIS_HOOK_POSTUNINSTALL
  MessageBox MB_YESNO "Delete all TidyBill data (database, settings, logos)?" /SD IDNO IDYES _delete_data IDNO _skip_data
  _delete_data:
    RMDir /r "$APPDATA\TidyBill"
  _skip_data:
!macroend
