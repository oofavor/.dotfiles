{
  pkgs,
  lib,
  config,
  ...
}:
let
  assoc = app: types: lib.genAttrs types (_: app);
in
{
  config = lib.mkIf config.ofavor.desktop.enable {
    # Lets GLib (Thunar, browsers) open Terminal=true entries such as nvim in ghostty.
    home.packages = [ pkgs.xdg-terminal-exec ];
    xdg.configFile."xdg-terminals.list".text = "com.mitchellh.ghostty.desktop\n";

    # Apps used to write this file themselves; now it is a read-only store link.
    xdg.configFile."mimeapps.list".force = true;
    # Home Manager writes the legacy location too.
    xdg.dataFile."applications/mimeapps.list".force = true;

    xdg.mimeApps = {
      enable = true;
      # Browser entries come from zen's setAsDefaultBrowser.
      defaultApplications =
        assoc "org.pwmt.zathura.desktop" [
          "application/pdf"
          "application/epub+zip"
          "image/vnd.djvu"
          "application/vnd.comicbook+zip"
        ]
        // assoc "org.nomacs.ImageLounge.desktop" [
          "image/png"
          "image/jpeg"
          "image/gif"
          "image/webp"
          "image/bmp"
          "image/tiff"
          "image/avif"
          "image/heic"
        ]
        // assoc "mpv.desktop" [
          "video/mp4"
          "video/x-matroska"
          "video/webm"
          "video/quicktime"
          "video/x-msvideo"
          "video/mpeg"
          "video/ogg"
          "audio/mpeg"
          "audio/flac"
          "audio/ogg"
          "audio/opus"
          "audio/wav"
          "audio/x-wav"
          "audio/mp4"
          "audio/aac"
        ]
        // assoc "org.gnome.FileRoller.desktop" [
          "application/zip"
          "application/x-tar"
          "application/x-compressed-tar"
          "application/x-xz-compressed-tar"
          "application/zstd"
          "application/x-zstd-compressed-tar"
          "application/x-7z-compressed"
          "application/vnd.rar"
        ]
        // assoc "nvim.desktop" [
          "text/plain"
          "text/markdown"
          "text/x-shellscript"
          "application/x-shellscript"
        ]
        // {
          "inode/directory" = "thunar.desktop";
          "x-scheme-handler/tg" = "org.telegram.desktop.desktop";
          "x-scheme-handler/tonsite" = "org.telegram.desktop.desktop";
          "x-scheme-handler/discord" = "vesktop.desktop";
          "x-scheme-handler/antigravity" = "antigravity.desktop";
          "x-scheme-handler/claude-cli" = "claude-code-url-handler.desktop";
        };
    };
  };
}
