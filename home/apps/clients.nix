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

            # QUIC is UDP, which Throne's auto-redirect can't hand to the kernel,
            # so every packet went through userspace twice. TCP measured far cheaper.
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
      (writeShellScriptBin "claude-web" ''
        set -euo pipefail
        export LANG="en_US.UTF-8"
        if [ "$#" -eq 0 ]; then
          set -- "https://claude.ai"
        fi
        exec isp-run ${config.programs.chromium.package}/bin/chromium \
          --user-data-dir="$HOME/.local/share/claude-chromium" \
          --class=claude-web \
          --proxy-server="socks5://127.0.0.1:10808" \
          --host-resolver-rules="MAP * ~NOTFOUND , EXCLUDE 127.0.0.1" \
          --force-webrtc-ip-handling-policy=disable_non_proxied_udp \
          --disable-quic \
          --lang=en-US \
          "$@"
      '')
    ];

    xdg.desktopEntries.claude-web = {
      name = "Claude Web";
      comment = "Isolated Chromium routed 100% through ISP";
      exec = "claude-web %U";
      icon = "chromium";
      categories = [
        "Network"
        "WebBrowser"
      ];
      startupNotify = true;
    };

    programs.chromium.enable = true;
  };
}
