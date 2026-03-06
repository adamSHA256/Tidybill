# THIS FILE IS AUTO-GENERATED. DO NOT MODIFY!!

# Copyright 2020-2023 Tauri Programme within The Commons Conservancy
# SPDX-License-Identifier: Apache-2.0
# SPDX-License-Identifier: MIT

-keep class com.tidybill.desktop.* {
  native <methods>;
}

-keep class com.tidybill.desktop.WryActivity {
  public <init>(...);

  void setWebView(com.tidybill.desktop.RustWebView);
  java.lang.Class getAppClass(...);
  java.lang.String getVersion();
}

-keep class com.tidybill.desktop.Ipc {
  public <init>(...);

  @android.webkit.JavascriptInterface public <methods>;
}

-keep class com.tidybill.desktop.RustWebView {
  public <init>(...);

  void loadUrlMainThread(...);
  void loadHTMLMainThread(...);
  void evalScript(...);
}

-keep class com.tidybill.desktop.RustWebChromeClient,com.tidybill.desktop.RustWebViewClient {
  public <init>(...);
}
