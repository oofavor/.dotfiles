{ pkgs, ... }:
let
  ispRun = pkgs.writeShellScriptBin "isp-run" ''
    set -euo pipefail

    fail() {
      local msg="$1"
      echo "isp-run: $msg" >&2
      if [ ! -t 2 ]; then
        ${pkgs.libnotify}/bin/notify-send -u critical "ISP Proxy Kill-Switch" "$msg" 2>/dev/null || true
      fi
      exit 1
    }

    if [ "$#" -eq 0 ]; then
      echo "Usage: isp-run <command> [args...]" >&2
      exit 2
    fi

    # 1. Enter the kernel-firewalled cgroup slice (blocks all non-loopback traffic in raw OUTPUT).
    if ! ${pkgs.gnugrep}/bin/grep -q 'isp-isolated\.slice' /proc/self/cgroup; then
      exec ${pkgs.systemd}/bin/systemd-run \
        --user \
        --scope \
        --quiet \
        --collect \
        --slice=isp-isolated.slice \
        -- "$0" "$@"
    fi

    # 2. Paranoid check that we are actually inside the isolated cgroup.
    if ! ${pkgs.gnugrep}/bin/grep -q 'isp-isolated\.slice' /proc/self/cgroup; then
      fail "Failed to enter isp-isolated.slice — refusing to start."
    fi

    # 3. Live kernel kill-switch self-test: direct outbound TCP MUST be dropped.
    if ${pkgs.coreutils}/bin/timeout 0.25 ${pkgs.bash}/bin/bash -c 'exec 3<>/dev/tcp/1.1.1.1/443' 2>/dev/null; then
      fail "KILL-SWITCH BREACH — direct outbound network is NOT blocked! Check: sudo systemctl status isp-killswitch-firewall"
    fi

    # 4. Require the dedicated local ISP proxy (HTTP :10809, SOCKS5 :10808).
    if ! ${pkgs.iproute2}/bin/ss -tlnH 'sport = :10809' | ${pkgs.gnugrep}/bin/grep -q . \
      || ! ${pkgs.iproute2}/bin/ss -tlnH 'sport = :10808' | ${pkgs.gnugrep}/bin/grep -q .; then
      fail "isp-proxy (127.0.0.1:10808/10809) is not listening — check: systemctl --user status isp-proxy"
    fi

    # 5. Live end-to-end exit IP verification + IP-synced timezone detection.
    cfg_dir="$HOME/.config/isp-proxy"
    expected_ip_file="$cfg_dir/expected-ip"
    tz_cache_file="$cfg_dir/timezone"

    if [ ! -r "$expected_ip_file" ]; then
      fail "Missing $expected_ip_file — refusing to start without expected ISP IP."
    fi
    expected_ip="$(${pkgs.coreutils}/bin/tr -d '[:space:]' < "$expected_ip_file")"
    if [ -z "$expected_ip" ]; then
      fail "$expected_ip_file is empty — refusing to start."
    fi

    actual_ip=""
    detected_tz=""

    for geo_url in "https://ipinfo.io/json" "https://ipapi.co/json"; do
      resp="$(${pkgs.curl}/bin/curl -fsS --max-time 4 -x "http://127.0.0.1:10809" "$geo_url" 2>/dev/null || true)"
      if [ -n "$resp" ]; then
        actual_ip="$(printf '%s' "$resp" | ${pkgs.jq}/bin/jq -r '.ip // empty' 2>/dev/null | ${pkgs.coreutils}/bin/tr -d '[:space:]' || true)"
        detected_tz="$(printf '%s' "$resp" | ${pkgs.jq}/bin/jq -r '.timezone // empty' 2>/dev/null | ${pkgs.coreutils}/bin/tr -d '[:space:]' || true)"
        if [ -n "$actual_ip" ]; then
          break
        fi
      fi
    done

    # Fallback to plain-text IP probe if both geo endpoints were rate-limited.
    if [ -z "$actual_ip" ]; then
      actual_ip="$(${pkgs.curl}/bin/curl -fsS --max-time 4 -x "http://127.0.0.1:10809" "https://api.ipify.org" 2>/dev/null | ${pkgs.coreutils}/bin/tr -d '[:space:]' || true)"
    fi

    if [ "$actual_ip" != "$expected_ip" ]; then
      fail "ISP proxy exit IP mismatch (expected '$expected_ip', got '$actual_ip') — refusing to start."
    fi

    # Validate detected IANA timezone against tzdata and cache it for fallback.
    resolved_tz=""
    if [ -n "$detected_tz" ] && [ -f "${pkgs.tzdata}/share/zoneinfo/$detected_tz" ]; then
      resolved_tz="$detected_tz"
      if [ -d "$cfg_dir" ]; then
        printf '%s\n' "$resolved_tz" > "$tz_cache_file" 2>/dev/null || true
        chmod 600 "$tz_cache_file" 2>/dev/null || true
      fi
    elif [ -r "$tz_cache_file" ]; then
      cached_tz="$(${pkgs.coreutils}/bin/tr -d '[:space:]' < "$tz_cache_file")"
      if [ -n "$cached_tz" ] && [ -f "${pkgs.tzdata}/share/zoneinfo/$cached_tz" ]; then
        resolved_tz="$cached_tz"
      fi
    fi

    if [ -n "$resolved_tz" ]; then
      export TZ="$resolved_tz"
    fi

    # 6. Export proxy environment and exec target command.
    export HTTP_PROXY="http://127.0.0.1:10809"
    export HTTPS_PROXY="http://127.0.0.1:10809"
    export http_proxy="http://127.0.0.1:10809"
    export https_proxy="http://127.0.0.1:10809"
    export ALL_PROXY="socks5h://127.0.0.1:10808"
    export all_proxy="socks5h://127.0.0.1:10808"
    export NO_PROXY="localhost,127.0.0.1,::1"
    export no_proxy="localhost,127.0.0.1,::1"

    exec "$@"
  '';

  ispCtl = pkgs.writeShellScriptBin "isp-ctl" ''
    export ISP_CTL_XRAY="${pkgs.xray}/bin/xray"
    export ISP_CTL_FZF="${pkgs.fzf}/bin/fzf"
    export ISP_CTL_CURL="${pkgs.curl}/bin/curl"
    export ISP_CTL_SYSTEMCTL="${pkgs.systemd}/bin/systemctl"
    export ISP_CTL_JOURNALCTL="${pkgs.systemd}/bin/journalctl"
    export ISP_CTL_TZDATA="${pkgs.tzdata}/share/zoneinfo"
    export PATH="${
      pkgs.lib.makeBinPath [
        pkgs.systemd
        pkgs.coreutils
        pkgs.bash
      ]
    }:$PATH"
    exec ${pkgs.python3}/bin/python3 ${./isp-ctl.py} "$@"
  '';
in
{
  home.packages = [
    ispRun
    ispCtl
  ];

  systemd.user.services.isp-proxy = {
    Unit = {
      Description = "Dedicated Xray VLESS proxy to residential ISP exit (SOCKS5 :10808, HTTP :10809)";
      After = [ "network-online.target" ];
      ConditionPathExists = "%h/.config/isp-proxy/config.json";
    };
    Service = {
      ExecStart = "${pkgs.xray}/bin/xray run -c %h/.config/isp-proxy/config.json";
      Restart = "always";
      RestartSec = 3;
    };
    Install = {
      WantedBy = [ "default.target" ];
    };
  };
}
