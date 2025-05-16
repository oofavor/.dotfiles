{
  pkgs,
  config,
  lib,
  ...
}:
{
  config = lib.mkIf config.ofavor.games.enable {
    home.packages = with pkgs; [
      (prismlauncher.override {
        # Add binary required by some mod
        additionalPrograms = [ ffmpeg ];

        # Change Java runtimes available to Prism Launcher
        jdks = [
          graalvm-ce
          zulu8
          zulu17
          zulu
        ];
      })
    ];
  };
}
