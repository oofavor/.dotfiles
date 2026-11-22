{ pkgs, username, ... }:
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

  # Reusable kernel cgroup v2 kill-switch for ISP-proxied apps (via `isp-run`).
  # Keeps isp-isolated.slice permanently pinned so its cgroup inode never
  # changes, then drops all non-loopback packets in raw OUTPUT (priority -300,
  # before conntrack and Throne's nat/mangle auto_redirect).
  boot.kernelModules = [ "xt_cgroup" ];
  users.users.${username}.linger = true;

  systemd.user.slices.isp-isolated = {
    description = "Network-isolated slice for ISP-proxied apps (loopback only)";
  };

  systemd.user.services.isp-isolated-anchor = {
    description = "Keep isp-isolated.slice cgroup inode pinned for firewall kill-switch";
    wantedBy = [ "default.target" ];
    serviceConfig = {
      Slice = "isp-isolated.slice";
      ExecStart = "${pkgs.coreutils}/bin/sleep infinity";
      Restart = "always";
      RestartSec = 1;
    };
  };

  systemd.services.isp-killswitch-firewall = {
    description = "Install iptables/ip6tables raw OUTPUT kill-switch for isp-isolated.slice";
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
      coreutils
      gnugrep
      gnused
      iptables
      systemd
    ];
    serviceConfig = {
      Type = "oneshot";
      RemainAfterExit = true;
    };
    script = ''
      set -euo pipefail
      uid="$(id -u "${username}")"
      # Systemd splits '-' in slice names into nested directories: isp-isolated.slice lives under isp.slice.
      rel_path="user.slice/user-''${uid}.slice/user@''${uid}.service/isp.slice/isp-isolated.slice"
      cg_dir="/sys/fs/cgroup/''${rel_path}"

      systemctl start "user@''${uid}.service" || true
      systemctl --machine="${username}@.host" --user daemon-reload || true
      systemctl --machine="${username}@.host" --user start isp-isolated-anchor.service || true

      for _ in $(seq 1 30); do
        [ -d "$cg_dir" ] && break
        sleep 0.5
      done
      if [ ! -d "$cg_dir" ]; then
        echo "isp-killswitch-firewall: cgroup directory $cg_dir does not exist" >&2
        exit 1
      fi

      for tool in iptables ip6tables; do
        loop_dst="127.0.0.1/32"
        if [ "$tool" = "ip6tables" ]; then
          loop_dst="::1/128"
        fi

        "$tool" -t raw -N ISP_KILLSWITCH 2>/dev/null || "$tool" -t raw -F ISP_KILLSWITCH
        "$tool" -t raw -A ISP_KILLSWITCH -d "$loop_dst" -j RETURN
        "$tool" -t raw -A ISP_KILLSWITCH -j DROP

        # Prune any existing or stale jumps to ISP_KILLSWITCH before inserting fresh inode match.
        while read -r line; do
          rule_spec="$(printf '%s\n' "$line" | sed 's/^-A OUTPUT //')"
          # shellcheck disable=SC2086
          "$tool" -t raw -D OUTPUT $rule_spec 2>/dev/null || true
        done < <("$tool" -t raw -S OUTPUT | grep ' -j ISP_KILLSWITCH$' || true)

        "$tool" -t raw -I OUTPUT 1 -m cgroup --path "$rel_path" -j ISP_KILLSWITCH
      done
    '';
  };
}
