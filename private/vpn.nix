{ pkgs, username, ... }:
let
  harpoPkg = pkgs.callPackage ./harpo/package.nix { };
in
{
  # nekoray (vpn)
  programs.throne.tunMode.enable = true;
  programs.throne.enable = true;

  # Clash Verge Rev (Mihomo)
  programs.clash-verge = {
    enable = true;
    tunMode = true;
    serviceMode = true;
  };

  # Kernel cgroup v2 kill-switch for harpo / harpo-run.
  # NixOS keeps /etc/systemd/system immutable, so the unit lives here and
  # delegates all firewall rule management to `harpo apply-firewall`.
  boot.kernelModules = [ "xt_cgroup" ];
  users.users.${username}.linger = true;

  systemd.user.slices.harpo-isolated = {
    description = "Network-isolated slice for Harpo sandboxed apps (loopback only)";
  };

  systemd.user.services.harpo-isolated-anchor = {
    description = "Keep harpo-isolated.slice cgroup inode pinned for Harpo kill-switch";
    wantedBy = [ "default.target" ];
    serviceConfig = {
      Slice = "harpo-isolated.slice";
      ExecStart = "${pkgs.coreutils}/bin/sleep infinity";
      Restart = "always";
      RestartSec = 1;
    };
  };

  systemd.services.harpo-killswitch = {
    description = "Install kernel cgroup v2 raw OUTPUT kill-switch for harpo-isolated.slice";
    after = [
      "firewall.service"
      "systemd-user-sessions.service"
    ];
    partOf = [ "firewall.service" ];
    wantedBy = [
      "multi-user.target"
      "firewall.service"
    ];
    path = with pkgs; [
      iptables
      kmod
      systemd
    ];
    serviceConfig = {
      Type = "oneshot";
      RemainAfterExit = true;
      ExecStart = "${harpoPkg}/bin/harpo apply-firewall --user ${username} --backend iptables";
    };
  };
}
