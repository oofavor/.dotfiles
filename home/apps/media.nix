{
  pkgs,
  lib,
  config,
  ...
}:
{
  config = lib.mkIf config.ofavor.desktop.enable {
    # PDF/EPUB/DjVu with vim keys; Stylix themes it.
    programs.zathura = {
      enable = true;
      options = {
        selection-clipboard = "clipboard";
        adjust-open = "width";
        recolor-keephue = true;
      };
    };

    programs.mpv = {
      enable = true;
      # mpris: media keys and the Quickshell media card control mpv too.
      scripts = with pkgs.mpvScripts; [ mpris ];
      config = {
        vo = "gpu-next";
        hwdec = "auto-safe";
        save-position-on-quit = true;
        keep-open = true;
        # Streams `mpv <youtube url>` through yt-dlp at most at the panel's height.
        ytdl-format = "bestvideo[height<=?1600]+bestaudio/best";
      };
    };

    programs.yt-dlp.enable = true;
  };
}
