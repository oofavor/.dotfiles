# Residential ISP Proxy & Kernel Kill-Switch Guide

This setup provides a reusable, fail-closed **ISP proxy + kernel firewall kill-switch** (`isp-run`) so any wrapped app (`claude`, `claude-web`, or ad-hoc commands) exits **100% through a residential ISP proxy** with zero leaks to either your real IP or your VPS IP — and automatically syncs `TZ` to the proxy IP's timezone.

> **Privacy note:** No proxy credentials, UUIDs, or server IPs are stored in this repository. Runtime secrets live in `~/.config/isp-proxy/` with `0600` permissions.

---

## Architecture

```text
┌─────────────────────────── Local NixOS Host ───────────────────────────┐
│                                                                        │
│  isp-run <cmd>  (used by `claude`, `claude-web`, or any CLI/GUI app)   │
│    └─ runs inside systemd cgroup: isp.slice/isp-isolated.slice         │
│         ├─ iptables/ip6tables raw OUTPUT (priority -300):              │
│         │    • allow 127.0.0.1/32 & ::1/128 ONLY                       │
│         │    • DROP everything else (IPv4/IPv6/TCP/UDP/DNS)            │
│         ├─ live self-tests: direct 1.1.1.1:443 blocked + exit IP match │
│         ├─ auto-syncs TZ from live IP geolocation (cached fallback)    │
│         └─ exports HTTP_PROXY/HTTPS_PROXY (:10809), ALL_PROXY (:10808) │
│                                                                        │
│  isp-proxy.service (standalone xray, independent of Throne)            │
│    • listens on 127.0.0.1:10808 (SOCKS5) & 127.0.0.1:10809 (HTTP)      │
│    • domainStrategy: AsIs (no local DNS resolution)                    │
│    • single outbound: VLESS Reality -> RU relay (NO direct outbound)   │
└──────────────────────────────────┬─────────────────────────────────────┘
                                   │ VLESS Reality (direct via geoip-ru,
                                   │ bypasses Throne TUN)
                                   ▼
┌────────────────────────────── RU Relay ────────────────────────────────┐
│  3x-ui / xray inbound -> 100% routed to foreign VPS ISP-only client    │
└──────────────────────────────────┬─────────────────────────────────────┘
                                   │ VLESS Reality
                                   ▼
┌───────────────────────────── Foreign VPS ──────────────────────────────┐
│  3x-ui / xray strict rule -> 100% routed to `isp-proxy` outbound       │
│  (strict rule: NO fallback to `direct` VPS IP if ISP proxy is down)    │
└──────────────────────────────────┬─────────────────────────────────────┘
                                   │ SOCKS5 + TCP keep-alive (15s/5s)
                                   ▼
┌────────────────────────────── ISP Proxy ───────────────────────────────┐
│  Residential/ISP exit IP (<EXPECTED_ISP_IP>) -> Internet               │
└────────────────────────────────────────────────────────────────────────┘
```

---

## Component Map

### Tracked in `.dotfiles`
| File | Role |
| :--- | :--- |
| `modules/services/vpn.nix` | Loads `xt_cgroup`, enables `linger` for the user, defines `isp-isolated.slice`, `isp-isolated-anchor.service`, and `isp-killswitch-firewall.service` (`iptables`/`ip6tables` `raw OUTPUT` `ISP_KILLSWITCH` chain). |
| `home/core/isp-proxy.nix` | Defines `isp-proxy.service` (`pkgs.xray`), the fail-closed `isp-run` launcher, and the `isp-ctl` interactive TUI. |
| `home/core/isp-ctl.py` | Interactive TUI (`isp-ctl`) to swap VLESS profiles from subscription URLs, single `vless://` links, or Throne DB with pre-flight `xray` validation, live exit-IP/ASN probe, and automatic rollback. |
| `home/core/default.nix` | Imports `./isp-proxy.nix`. |
| `home/dev/claude.nix` | Wraps `claude-code` via `isp-run` (plus telemetry/nonessential-traffic opt-outs), enforcing isolation across all shells, editors, and URI handlers. |
| `home/apps/clients.nix` | `claude-web` launcher: runs isolated Chromium via `isp-run` with `--proxy-server=socks5://127.0.0.1:10808`, `--host-resolver-rules`, `--disable-quic`, and WebRTC UDP disabled. |

### Untracked Local Files (`0700` dir / `0600` files)
| Path | Contents |
| :--- | :--- |
| `~/.config/isp-proxy/config.json` | Xray client config: `socks-in` (`127.0.0.1:10808`), `http-in` (`127.0.0.1:10809`), single `isp-exit` VLESS Reality outbound to the RU relay IP with `sockopt` TCP keep-alives (`tcpKeepAliveIdle: 15`, `tcpKeepAliveInterval: 5`, `tcpNoDelay: true`), `domainStrategy: "AsIs"`. |
| `~/.config/isp-proxy/expected-ip` | Single line containing the expected residential ISP proxy exit IP. Verified on every `isp-run` launch. |
| `~/.config/isp-proxy/timezone` | Auto-updated IANA timezone cache (e.g. `Europe/Berlin`) synced from `ipinfo.io` / `ipapi.co` through the proxy, used as fallback if geo endpoints are rate-limited. |

### VPS-Side Tuning (outside `.dotfiles`)
- **Foreign VPS**:
  - `/etc/x-ui/x-ui.db` (`xrayTemplateConfig` → `isp-proxy` outbound) has `streamSettings.sockopt` set to `tcpKeepAliveIdle: 15`, `tcpKeepAliveInterval: 5`, `tcpNoDelay: true`. Backup at `/etc/x-ui/x-ui.db.bak-isp-tuning`.
  - `/etc/sysctl.d/99-xray-tuning.conf` sets `tcp_keepalive_time=60`, `tcp_keepalive_intvl=10`, `tcp_keepalive_probes=6`.
- **RU Relay**:
  - `/etc/sysctl.d/99-xray-keepalive.conf` sets `tcp_keepalive_time=60`, `tcp_keepalive_intvl=10`, `tcp_keepalive_probes=6`.

---

## Reusing `isp-run` for Other Apps

Any CLI or Electron/Chromium app can reuse the exact same kill-switch, ISP proxy, and synced timezone:

```bash
# Run any one-off command through the ISP proxy + kill-switch
isp-run curl -sS https://ipinfo.io/json
isp-run env | grep -E 'TZ|PROXY'

# Example: wrapping an Electron app (like Claude Desktop)
isp-run claude-desktop \
  --proxy-server="socks5://127.0.0.1:10808" \
  --host-resolver-rules="MAP * ~NOTFOUND , EXCLUDE 127.0.0.1" \
  --force-webrtc-ip-handling-policy=disable_non_proxied_udp \
  --disable-quic
```

---

## Why Specific Design Choices Were Made

1. **`raw OUTPUT` (priority `-300`) instead of `filter OUTPUT`**:
   Throne runs `sing-box` with `vpn_auto_redirect = true`, which uses nftables `route`/`mangle` (overwriting `meta mark`) and `nat OUTPUT` (rewriting destination to `127.0.0.1` on `lo`). Placing `ISP_KILLSWITCH` in `raw OUTPUT` runs **before** conntrack and Throne's redirect, so non-loopback traffic from `isp-isolated.slice` can never be hijacked into Throne.
2. **Allowing `127.0.0.1/32` and `::1/128` (not `127.0.0.0/8`)**:
   Allows wrapped apps to reach the local ISP proxy (`:10809` / `:10808`) and local dev servers/MCP (`localhost:3000`, Postgres, etc.) while still blocking `127.0.0.53` (`systemd-resolved`) and all external IPv4/IPv6/UDP/DNS targets.
3. **Nested slice path `isp.slice/isp-isolated.slice`**:
   systemd turns `-` in `.slice` unit names into nested cgroup directories, so `isp-isolated.slice` lives at `user.slice/user-<UID>.slice/user@<UID>.service/isp.slice/isp-isolated.slice`.
4. **Why `isp-isolated-anchor.service` (`sleep infinity`) exists**:
   Linux `xt_cgroup` (`iptables -m cgroup --path`) pins the cgroup inode at rule-insertion time. If a slice becomes empty, systemd removes its directory and recreates it later with a new inode, breaking the iptables match. `linger = true` + the anchor service keep the directory and inode permanently alive for the entire boot.
5. **IP-synced timezone**:
   On every launch, `isp-run` queries `ipinfo.io/json` (fallback `ipapi.co/json`, then `api.ipify.org`) through `127.0.0.1:10809`, checks `.ip == expected-ip`, validates `.timezone` against `tzdata`, caches it to `~/.config/isp-proxy/timezone`, and exports `TZ`. Rotating to a proxy in another country automatically updates `TZ` for all wrapped apps.
6. **HTTP `:10809` + SOCKS5 `:10808`**:
   Bun/Node runtimes (like Claude Code) only support `http://`/`https://` in `HTTPS_PROXY`, while Chromium/Electron and Firefox/Zen containers use `socks5://127.0.0.1:10808`.

---

## Verification & Health Checks

```bash
# 1. Check services
systemctl --user status isp-proxy isp-isolated-anchor --no-pager
sudo systemctl status isp-killswitch-firewall --no-pager

# 2. Inspect kernel raw OUTPUT rules
sudo iptables -t raw -L OUTPUT -n -v
sudo iptables -t raw -L ISP_KILLSWITCH -n -v
sudo ip6tables -t raw -L ISP_KILLSWITCH -n -v

# 3. Verify direct outbound is kernel-dropped inside the slice (MUST time out, exit 28)
systemd-run --user --scope --quiet --collect --slice=isp-isolated.slice -- \
  curl --noproxy '*' --max-time 2 -sS https://api.ipify.org

# 4. Verify isp-run passes self-tests, exits via ISP IP, and syncs TZ
isp-run sh -c 'echo "TZ=$TZ"; curl -fsS https://ipinfo.io/json'

# 5. Verify wrapped claude
claude --version
```

---

## Maintenance & Troubleshooting

### Swapping Profiles / Subscriptions with `isp-ctl` (Recommended)
Run the interactive TUI:
```bash
isp-ctl
```
Or jump directly from the CLI:
```bash
isp-ctl sub "https://your-subscription-url"   # Fetch sub, fuzzy-pick VLESS profile, pre-flight test & swap
isp-ctl uri "vless://..."                     # Pre-flight test & swap a single VLESS link
isp-ctl status                                # Live health check (kill-switch, exit IP, ASN, TZ, latency)
```
Before touching `~/.config/isp-proxy/config.json`, `isp-ctl` validates with `xray run -test`, runs a live probe on a temporary port showing the candidate's real exit IP / ASN / country / timezone, backs up `config.json.bak`, and automatically rolls back if `isp-proxy.service` fails to restart.

### Rotating the ISP Proxy IP (Manual)
1. Update the `isp-proxy` SOCKS5 server credentials/IP in `3x-ui` on the foreign VPS (keep `streamSettings.sockopt` TCP keep-alives on the `isp-proxy` outbound).
2. Update the expected IP locally (timezone auto-syncs on next launch):
   ```bash
   printf '<NEW_ISP_IP>\n' > ~/.config/isp-proxy/expected-ip
   chmod 600 ~/.config/isp-proxy/expected-ip
   ```
3. Verify with `isp-run sh -c 'echo "TZ=$TZ"; curl -fsS https://ipinfo.io/json'`.

### Updating RU Relay VLESS Credentials
1. Edit `~/.config/isp-proxy/config.json` (keep `0600` permissions, `inbounds` on `10808`/`10809`, `domainStrategy: "AsIs"`, `sockopt` keep-alives, and **no `direct` outbound**).
2. Validate and restart:
   ```bash
   xray_bin="$(systemctl --user show -p ExecStart --value isp-proxy | awk '{print $2}' | tr -d ';')"
   "$xray_bin" run -test -c ~/.config/isp-proxy/config.json
   systemctl --user restart isp-proxy
   ```

### `isp-run: KILL-SWITCH BREACH — direct outbound network is NOT blocked!`
The `raw OUTPUT` cgroup rule is missing or pinned to a stale inode. Fix:
```bash
systemctl --user restart isp-isolated-anchor
sudo systemctl restart isp-killswitch-firewall
```

### `isp-run: ISP proxy exit IP mismatch`
1. Check local proxy logs: `journalctl --user -u isp-proxy -n 50 --no-pager`
2. Test manually: `curl -v --max-time 6 -x http://127.0.0.1:10809 https://api.ipify.org`
3. Check `x-ui` on `ru` and `aeza`, and test `curl -x socks5h://127.0.0.1:10808 https://api.ipify.org` on `aeza`.

---

## Known Boundaries

- **Browser OAuth (`claude /login`)**: `xdg-open` opens your default browser (Zen, routed via Throne), **not** `claude-web`. Paste Claude login URLs into `claude-web` (or a proxied Zen Container) so auth flows also use the ISP proxy.
- **Localhost access**: Any service listening on `127.0.0.1` is reachable from `isp-isolated.slice` by design (so `:10808`/`:10809` and local dev servers/MCP work).
- **Reboots & updates**: Survives reboots, firewall reloads (`PartOf=firewall.service`), Throne restarts/profile switches, and `just update` / `just switch` automatically.
