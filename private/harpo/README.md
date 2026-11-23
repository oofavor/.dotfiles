# harpo

Cross-platform (Linux & Windows) **Proxy, Update-Proof App Sandbox & Kill-Switch Manager** written in pure Go (`CGO_ENABLED=0`), featuring a minimal, airy, terminal-native TUI (inherits your terminal theme), zero automatic background network traffic, on-demand multi-domain probing, dynamic PATH shims, built-in `harpo-run`, and a 6-suite verification suite.

## Highlights
- **Zero Automatic External Traffic**: Launching `harpo` reads only local state (`systemd`/`Scheduled Task`, `config.json`, processes, logs). Live exit-IP checks and domain probes run **strictly when you press a key**.
- **Minimal, Terminal-Native 5-Tab UI (`harpo`)**:
  1. **Dashboard**: Top-to-bottom Tunnel & Shield summary (`i` to probe live exit IP) + scrollable Live Logs (`v` switches bottom feed to active isolated processes).
  2. **Apps**: Update-proof wrapped apps manager (`apps.json`) with auto-discovery of installed desktop apps, one-key pause/resume, and `.desktop` (Linux) / Start Menu `.lnk` (Windows) launchers.
  3. **Exits**: Compact Source strip (`Throne DB`, saved subs, URL, `vless://`) above a full-width VLESS nodes list. Pressing `Enter` runs local `xray -test` first; press `p` to optionally probe exit IP or `Enter` again to activate.
  4. **Probe**: On-demand Exit IP/Kill-Switch check (`i`) + customizable Domain Reachability table (`targets.json`, seeded with Anthropic, Google Gemini, OpenAI, YouTube, GitHub, Cloudflare). Press `Enter` to probe one domain, `p` to probe all concurrently, `m` to switch HTTP `:10809` / SOCKS5h `:10808`, `a`/`d` to add/remove targets.
  5. **System**: Verification Tests (`t`), Setup Checklist (`s`), and Full Logs/Restarts (`l`).
- **Update-Proof App Shims**:
  - **Linux**: Symlinks in `~/.local/share/harpo/shims/` dynamically resolve the real binary on every run.
  - **Windows**: Zero-privilege `.cmd` shims in `%LOCALAPPDATA%\harpo\shims\` + Start Menu `.lnk` shortcuts in `Start Menu\Programs\Harpo\`.

## Building
```bash
# Linux (native)
make build

# Windows (amd64 & arm64 cross-compile from Linux/macOS)
make build-windows
make build-windows-arm64
```

## Windows Quick Start
1. Place `harpo.exe` and `xray.exe` (from Xray-core Windows releases) in the same folder (e.g. `C:\Tools\harpo\`) and add that folder to `PATH` (or run `.\harpo.exe` directly).
2. Run `harpo setup` (no Admin required):
   - Registers a hidden per-user logon Scheduled Task (`HarpoProxy`) that keeps `xray.exe` running silently (zero console flash, logs in `%LOCALAPPDATA%\harpo\logs\xray.log`).
   - Prepends `%LOCALAPPDATA%\harpo\shims` to your Windows User `PATH`.
3. Open `harpo` → `3 Exits` → load Throne DB / subscription / `vless://` → activate a node (zero UAC prompts!).
4. Wrap apps (`2 Apps` → `a`, or CLI):
   ```powershell
   harpo wrap claude --mode cli --env DISABLE_TELEMETRY=1 --env DO_NOT_TRACK=1
   harpo wrap claude-web --mode chromium --cmd chrome --desktop --isolated-profile --default-arg https://claude.ai
   ```
5. **Kill-switch (dual mode)**:
   - **Soft mode (works out of the box, no Admin)**: `harpo launch` / `harpo run` fails closed before launch if proxy `:10808/:10809` is down or live exit IP != `expected-ip`, and injects `HTTP_PROXY`/`HTTPS_PROXY`/`ALL_PROXY`, synced `TZ`, and anti-leak Chromium flags.
   - **Hard Windows Firewall kill-switch (recommended)**: In an **Administrator** terminal, run:
     ```powershell
     harpo setup-firewall          # Enable WFP outbound block rules (loopback 127.0.0.1 / ::1 only) for wrapped .exe apps
     harpo apply-firewall --clear  # Remove all Harpo Windows Firewall rules
     ```
   - Use `harpo run --strict <cmd>` (or `HARPO_STRICT=1`) to refuse launching without an active WFP rule, or `--soft` / `--no-firewall` per app.
6. If running alongside Throne / sing-box TUN on Windows, add `xray.exe` to Throne's Direct `process_name` routing rules so direct nodes bypass TUN cleanly.
