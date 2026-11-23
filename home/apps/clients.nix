{
  pkgs,
  lib,
  config,
  inputs,
  ...
}:
{
  imports = [ inputs.zen-browser.homeModules.twilight ];

  config = lib.mkIf config.ofavor.desktop.enable {
    stylix.targets.zen-browser.enable = false;

    programs.zen-browser = {
      enable = true;
      setAsDefaultBrowser = true;

      # The profile had fallen back to software WebRender (CPU painting, no VA-API).
      # Force GPU compositing and hardware video decode; the iGPU does AV1/VP9/HEVC/H.264.
      policies.Preferences =
        lib.mapAttrs
          (_: value: {
            Value = value;
            Status = "user";
          })
          {
            "gfx.webrender.all" = true;
            "gfx.webrender.software" = false;
            "gfx.webrender.software.opengl" = false;
            "media.ffmpeg.vaapi.enabled" = true;
            "media.hardware-video-decoding.force-enabled" = true;

            "network.http.http3.enable" = false;
            # Session snapshot every 60s instead of 15s: fewer SSD wakeups.
            "browser.sessionstore.interval" = 60000;
          };
    };

    home.packages = with pkgs; [
      vesktop
      obsidian
      qbittorrent
      telegram-desktop
    ];

    programs.chromium.enable = true;
  };
}
