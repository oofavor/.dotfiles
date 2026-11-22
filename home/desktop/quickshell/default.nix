{
  pkgs,
  lib,
  config,
  ...
}:
let
  colors = config.lib.stylix.colors.withHashtag;
  fonts = config.stylix.fonts;

  wallpaper = ../../../assets/background.jpg;

  # Softened copy of the wallpaper for niri's overview backdrop.
  backdrop = pkgs.runCommand "wallpaper-backdrop.jpg" { nativeBuildInputs = [ pkgs.imagemagick ]; } ''
    magick ${wallpaper} -resize 25% -blur 0x12 -resize 1920x1080^ -fill black -colorize 25% $out
  '';

  # "😀<TAB>grinning face" per line for the launcher's emoji mode (skin tone variants left out).
  emojiList = pkgs.runCommand "launcher-emoji.tsv" { } ''
    ${lib.getExe pkgs.gawk} -F'# ' '/; fully-qualified/ && !/skin tone/ {
      split($2, a, " "); name = $2; sub(/^[^ ]+ E[0-9.]+ /, "", name); print a[1] "\t" name
    }' ${pkgs.unicode-emoji}/share/unicode/emoji/emoji-test.txt > $out
  '';

  qmlList = xs: "[" + lib.concatMapStringsSep ", " (x: ''"${x}"'') xs + "]";

  # Everything the QML reads from the system lives in this one generated singleton.
  generated = ''
    pragma Singleton

    import QtQuick
    import Quickshell

    Singleton {
    ${lib.concatMapStrings (n: "    readonly property color ${n}: \"${colors.${n}}\"\n") (
      map (i: "base0${lib.toHexString i}") (lib.range 0 15)
    )}
        readonly property string fontSans: "${fonts.sansSerif.name}"
        readonly property string fontMono: "${fonts.monospace.name}"

        readonly property string wallpaper: "file://${wallpaper}"
        readonly property string backdrop: "file://${backdrop}"

        readonly property var lockCommand: ${qmlList [ config.ofavor.desktop.lockCommand ]}
        readonly property var terminal: ${qmlList [ "ghostty" ]}

        // Launcher helpers (modules/panels/Launcher.qml).
        readonly property string qalc: "${lib.getExe' pkgs.libqalculate "qalc"}"
        readonly property string wlCopy: "${lib.getExe' pkgs.wl-clipboard "wl-copy"}"
        readonly property string cliphist: "${lib.getExe config.services.cliphist.package}"
        readonly property string emojiFile: "${emojiList}"
        // "+ text" in the launcher appends "- text" here.
        readonly property string notesFile: "${config.home.homeDirectory}/obsidian/Base/DUMP.md"

        // Built-in panel whose refresh rate services/Display.qml manages.
        readonly property string internalOutput: "eDP-1"
        // Brightness ceiling on unplug (modules/power/BatteryBrightness.qml); ~0.04 W per percent.
        readonly property real batteryBrightnessCap: 0.35
    }
  '';
in
{
  config = lib.mkIf config.ofavor.desktop.enable {
    # Runs as a user service so it restarts if it crashes.
    programs.quickshell = {
      enable = true;
      systemd.enable = true;
    };
    systemd.user.services.quickshell.Unit.PartOf = [ "graphical-session.target" ];

    home.packages = with pkgs; [
      wlsunset # night light
      libnotify
      playerctl
      networkmanagerapplet # nm-connection-editor
      pwvucontrol
    ];

    xdg.configFile."quickshell" = {
      source = ./config;
      recursive = true;
    };
    xdg.configFile."quickshell/common/Generated.qml".text = generated;
  };
}
