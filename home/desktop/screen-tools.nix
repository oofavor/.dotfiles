{
  pkgs,
  lib,
  config,
  ...
}:
let
  niri = config.programs.niri.package;

  # Region screenshot opened in satty for arrows, blur and crop; Enter copies it.
  annotate = pkgs.writeShellApplication {
    name = "screenshot-annotate";
    runtimeInputs = with pkgs; [
      grim
      slurp
      satty
    ];
    text = ''
      region=$(slurp -d) || exit 0
      grim -g "$region" - | satty --filename -
    '';
  };

  # Toggles a recording of the focused monitor (with desktop audio) into ~/Videos/Recordings.
  # gpu-screen-recorder comes from the NixOS module, which adds its setcap helper.
  # Start/stop is reported to quickshell, which shows a timer in the bar.
  record = pkgs.writeShellApplication {
    name = "screen-record";
    runtimeInputs = [
      niri
      pkgs.coreutils
      pkgs.jq
      pkgs.libnotify
      config.programs.quickshell.package
    ];
    text = ''
      shell() { quickshell ipc call recorder "$1" >/dev/null 2>&1 || true; }

      pidfile="''${XDG_RUNTIME_DIR:-/tmp}/screen-record.pid"
      if [ -f "$pidfile" ] && kill -0 "$(cat "$pidfile")" 2>/dev/null; then
        kill -INT "$(cat "$pidfile")"
        exit 0
      fi

      dir="$HOME/Videos/Recordings"
      mkdir -p "$dir"
      file="$dir/$(date +%Y-%m-%d_%H-%M-%S).mp4"
      output=$(niri msg --json focused-output | jq -r .name)

      gpu-screen-recorder -w "$output" -f 60 -a default_output -o "$file" &
      echo $! > "$pidfile"
      shell started
      notify-send -a "Screen recorder" -i media-record "Recording $output" "Mod+Shift+R again to stop"

      status=0
      wait $! || status=$?
      rm -f "$pidfile"
      shell stopped
      if [ -s "$file" ]; then
        notify-send -a "Screen recorder" -i video-x-generic "Recording saved" "$file"
      else
        notify-send -a "Screen recorder" -u critical "Recording failed" "gpu-screen-recorder exited with $status"
      fi
    '';
  };

  pickColor = pkgs.writeShellApplication {
    name = "pick-color";
    runtimeInputs = [
      niri
      pkgs.gawk
      pkgs.wl-clipboard
      pkgs.libnotify
    ];
    text = ''
      hex=$(niri msg pick-color | awk '/^Hex:/ { print $2 }')
      [ -n "$hex" ] || exit 0
      printf '%s' "$hex" | wl-copy
      notify-send -a "Color picker" "$hex" "Copied to clipboard"
    '';
  };

  # Text from a screen region (English + Russian) straight to the clipboard.
  ocr = pkgs.writeShellApplication {
    name = "ocr-region";
    runtimeInputs = with pkgs; [
      grim
      slurp
      (tesseract.override {
        enableLanguages = [
          "eng"
          "rus"
        ];
      })
      wl-clipboard
      libnotify
    ];
    text = ''
      region=$(slurp) || exit 0
      text=$(grim -g "$region" - | tesseract stdin stdout -l eng+rus 2>/dev/null)
      if [ -z "''${text//[[:space:]]/}" ]; then
        notify-send -a "OCR" "No text found"
        exit 0
      fi
      printf '%s' "$text" | wl-copy
      notify-send -a "OCR" "Copied text" "$(printf '%s' "$text" | head -c 200)"
    '';
  };
in
{
  config = lib.mkIf config.ofavor.desktop.enable {
    home.packages = [
      annotate
      record
      pickColor
      ocr
    ];

    programs.satty = {
      enable = true;
      settings.general = {
        early-exit = [ "all" ];
        initial-tool = "arrow";
        corner-roundness = 12;
        copy-command = "wl-copy";
        output-filename = "~/Pictures/Screenshots/%Y-%m-%d_%H-%M-%S-annotated.png";
        actions-on-enter = [ "save-to-clipboard" ];
        actions-on-escape = [ "exit" ];
      };
    };

    programs.niri.settings = {
      binds = {
        "Mod+Shift+a".action.spawn = [ (lib.getExe annotate) ];
        "Mod+Shift+r".action.spawn = [ (lib.getExe record) ];
        "Mod+Shift+p".action.spawn = [ (lib.getExe pickColor) ];
        "Mod+Shift+t".action.spawn = [ (lib.getExe ocr) ];
      };
      window-rules = [
        {
          matches = [ { app-id = "^com\\.gabm\\.satty$"; } ];
          open-floating = true;
        }
      ];
    };
  };
}
