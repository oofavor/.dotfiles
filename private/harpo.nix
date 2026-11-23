{
  pkgs,
  lib,
  config,
  ...
}:
let
  harpoPkg = pkgs.callPackage ./harpo/package.nix { };
in
{
  home.packages = [
    harpoPkg
    pkgs.xray
  ];

  # Ensure update-proof harpo shims take priority ahead of ~/.local/bin and ~/.npm-global/bin.
  home.sessionPath = lib.mkBefore [
    "$HOME/.local/share/harpo/shims"
  ];

  # Keep shims & managed .desktop overrides synced automatically on every Home Manager switch.
  home.activation.harpoSyncShims = lib.hm.dag.entryAfter [ "writeBoundary" ] ''
    if [ -r "$HOME/.config/harpo/apps.json" ]; then
      run ${harpoPkg}/bin/harpo shims sync || true
    fi
  '';

  systemd.user.services.harpo-proxy = {
    Unit = {
      Description = "Harpo Xray VLESS proxy (SOCKS5 :10808, HTTP :10809)";
      After = [ "network-online.target" ];
      ConditionPathExists = "%h/.config/harpo/config.json";
    };
    Service = {
      ExecStart = "${pkgs.xray}/bin/xray run -c %h/.config/harpo/config.json";
      Restart = "always";
      RestartSec = 3;
    };
    Install = {
      WantedBy = [ "default.target" ];
    };
  };

  xdg.mimeApps.defaultApplications = lib.mkIf config.ofavor.desktop.enable {
    "x-scheme-handler/throne" = "throne-url-handler.desktop";
  };
}
