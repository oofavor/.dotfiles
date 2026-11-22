{
  pkgs,
  lib,
  config,
  ...
}:
let
  # Quickshell lock screen (modules/lock), or swaylock when the shell isn't running.
  lockSession = pkgs.writeShellApplication {
    name = "lock-session";
    runtimeInputs = [
      config.programs.quickshell.package
      pkgs.coreutils
    ];
    text = ''
      if quickshell ipc call lock lock >/dev/null 2>&1; then
        # Return once every output is covered (max 2 s), so suspend never shows the desktop.
        for _ in $(seq 40); do
          [ "$(quickshell ipc call lock isSecure 2>/dev/null)" = true ] && exit 0
          sleep 0.05
        done
        exit 0
      fi
      exec ${lib.getExe config.programs.swaylock.package} -f
    '';
  };
in
{
  options.ofavor.desktop.lockCommand = lib.mkOption {
    type = lib.types.str;
    readOnly = true;
    default = lib.getExe lockSession;
    description = "Locks the session; used by swayidle, keybinds and the power menu.";
  };

  config = lib.mkIf config.ofavor.desktop.enable {
    home.packages = [ lockSession ];
  };
}
