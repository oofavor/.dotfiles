{
  lib,
  username,
  ...
}:
{
  imports = [
    ./core
    ./dev
    ./desktop
    ./apps
  ];

  config = {
    home.username = username;
    home.homeDirectory = "/home/${username}";
    programs.home-manager.enable = true;

    # gtk.gtk4.theme = config.gtk.theme;
  };

  options.ofavor = {
    desktop.enable = lib.mkEnableOption "desktop environment";

    games.enable = lib.mkEnableOption "gaming packages";

    dev.android.enable = lib.mkEnableOption "Android SDK, NDK, emulator and JDK";

    dev.langs = lib.mkOption {
      type = lib.types.listOf (
        lib.types.enum [
          "zig"
          "rust"
          "node"
          "go"
          "prolog"
          "python"
        ]
      );
      default = [
        "zig"
        "rust"
        "node"
        "go"
        "prolog"
        "python"
      ];
      description = "Development languages to install";
    };
  };
}
