#!/usr/bin/env python3
"""isp-ctl: Interactive TUI for managing ~/.config/isp-proxy (Xray + kill-switch)."""

from __future__ import annotations

import base64
import ipaddress
import json
import os
import shutil
import socket
import sqlite3
import subprocess
import sys
import tempfile
import time
import urllib.parse
from pathlib import Path

CFG_DIR = Path.home() / ".config" / "isp-proxy"
CFG_PATH = CFG_DIR / "config.json"
BACKUP_PATH = CFG_DIR / "config.json.bak"
EXPECTED_IP_PATH = CFG_DIR / "expected-ip"
TZ_PATH = CFG_DIR / "timezone"
SUBS_PATH = CFG_DIR / "subscriptions.json"
ACTIVE_PROFILE_PATH = CFG_DIR / "active-profile"
THRONE_DB = Path.home() / ".config" / "Throne" / "config" / "throne.db"

XRAY_BIN = os.environ.get("ISP_CTL_XRAY", "xray")
FZF_BIN = os.environ.get("ISP_CTL_FZF", "fzf")
CURL_BIN = os.environ.get("ISP_CTL_CURL", "curl")
SYSTEMCTL_BIN = os.environ.get("ISP_CTL_SYSTEMCTL", "systemctl")
JOURNALCTL_BIN = os.environ.get("ISP_CTL_JOURNALCTL", "journalctl")
TZDATA_DIR = os.environ.get("ISP_CTL_TZDATA", "/usr/share/zoneinfo")

SOCKS_PORT = 10808
HTTP_PORT = 10809

SOCKOPT_DEFAULT = {
    "tcpKeepAliveIdle": 15,
    "tcpKeepAliveInterval": 5,
    "tcpNoDelay": True,
}

# ANSI colors
BOLD = "\033[1m"
DIM = "\033[2m"
RED = "\033[31m"
GREEN = "\033[32m"
YELLOW = "\033[33m"
CYAN = "\033[36m"
RESET = "\033[0m"


def ensure_cfg_dir() -> None:
    CFG_DIR.mkdir(parents=True, exist_ok=True)
    os.chmod(CFG_DIR, 0o700)


def write_secret_text(path: Path, text: str) -> None:
    ensure_cfg_dir()
    fd, tmp_name = tempfile.mkstemp(dir=CFG_DIR, prefix=f".{path.name}.")
    try:
        os.chmod(tmp_name, 0o600)
        with os.fdopen(fd, "w", encoding="utf-8") as f:
            f.write(text)
        os.replace(tmp_name, path)
    except Exception:
        if os.path.exists(tmp_name):
            os.unlink(tmp_name)
        raise


def read_text(path: Path) -> str:
    try:
        return path.read_text(encoding="utf-8").strip()
    except OSError:
        return ""


def is_ip_literal(host: str) -> bool:
    try:
        ipaddress.ip_address(host)
        return True
    except ValueError:
        return False


def resolve_ipv4(host: str) -> str | None:
    try:
        infos = socket.getaddrinfo(host, None, family=socket.AF_INET, type=socket.SOCK_STREAM)
        if infos:
            return str(infos[0][4][0])
    except OSError:
        return None
    return None


def find_free_port() -> int:
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
        s.bind(("127.0.0.1", 0))
        return int(s.getsockname()[1])


def svc_active(unit: str, user: bool = True) -> str:
    cmd = [SYSTEMCTL_BIN]
    if user:
        cmd.append("--user")
    cmd.extend(["is-active", unit])
    res = subprocess.run(cmd, capture_output=True, text=True, check=False)
    return res.stdout.strip() or "unknown"


def load_current_summary() -> dict[str, str]:
    info = {
        "name": read_text(ACTIVE_PROFILE_PATH) or "(unnamed)",
        "server": "none",
        "security": "-",
        "sni": "-",
        "expected_ip": read_text(EXPECTED_IP_PATH) or "(unset)",
        "tz": read_text(TZ_PATH) or "(auto)",
        "proxy_state": svc_active("isp-proxy.service", user=True),
        "fw_state": svc_active("isp-killswitch-firewall.service", user=False),
    }
    if CFG_PATH.is_file():
        try:
            cfg = json.loads(CFG_PATH.read_text(encoding="utf-8"))
            ob = (cfg.get("outbounds") or [{}])[0]
            host, port = extract_host_port(ob)
            if host:
                info["server"] = f"{host}:{port}"
            ss = ob.get("streamSettings") or {}
            info["security"] = ss.get("security") or "none"
            rs = ss.get("realitySettings") or ss.get("tlsSettings") or {}
            info["sni"] = rs.get("serverName") or "-"
        except Exception:
            pass
    return info


def extract_host_port(ob: dict) -> tuple[str, int]:
    settings = ob.get("settings") or {}
    if "address" in settings:
        return str(settings.get("address", "")), int(settings.get("port", 0))
    vnext = settings.get("vnext") or []
    if vnext:
        return str(vnext[0].get("address", "")), int(vnext[0].get("port", 0))
    return "", 0


def set_outbound_host(ob: dict, new_host: str) -> dict:
    ob = json.loads(json.dumps(ob))
    settings = ob.setdefault("settings", {})
    if "address" in settings:
        settings["address"] = new_host
    elif settings.get("vnext"):
        settings["vnext"][0]["address"] = new_host
    return ob


def build_xray_config(outbound: dict, socks_port: int = SOCKS_PORT, http_port: int = HTTP_PORT) -> dict:
    ob = json.loads(json.dumps(outbound))
    ob["tag"] = "isp-exit"
    ss = ob.setdefault("streamSettings", {})
    so = ss.setdefault("sockopt", {})
    for k, v in SOCKOPT_DEFAULT.items():
        so.setdefault(k, v)

    return {
        "log": {"loglevel": "warning"},
        "inbounds": [
            {
                "tag": "socks-in",
                "listen": "127.0.0.1",
                "port": socks_port,
                "protocol": "socks",
                "settings": {"auth": "noauth", "udp": False},
            },
            {
                "tag": "http-in",
                "listen": "127.0.0.1",
                "port": http_port,
                "protocol": "http",
                "settings": {},
            },
        ],
        "outbounds": [ob],
        "routing": {"domainStrategy": "AsIs", "rules": []},
    }


def parse_vless_uri(uri: str) -> dict:
    uri = uri.strip()
    if not uri.startswith("vless://"):
        raise ValueError("URI does not start with vless://")

    parsed = urllib.parse.urlsplit(uri)
    uuid = urllib.parse.unquote(parsed.username or "")
    host = parsed.hostname or ""
    port = parsed.port or 443
    name = urllib.parse.unquote(parsed.fragment).strip() or f"{host}:{port}"
    if not uuid or not host:
        raise ValueError("Missing UUID or host in VLESS URI")

    qs = urllib.parse.parse_qs(parsed.query, keep_blank_values=True)

    def q(key: str, default: str = "") -> str:
        vals = qs.get(key)
        return vals[0] if vals else default

    net_type = q("type", "tcp").lower()
    if net_type == "tcp":
        net_type = "raw"
    security = q("security", "none").lower()
    encryption = q("encryption", "none") or "none"
    flow = q("flow", "")

    stream_settings: dict = {
        "network": net_type,
        "security": security,
        "sockopt": dict(SOCKOPT_DEFAULT),
    }

    if security == "reality":
        pbk = q("pbk")
        if not pbk:
            raise ValueError("Reality VLESS link is missing 'pbk' (public key)")
        rs: dict = {
            "fingerprint": q("fp", "firefox") or "firefox",
            "password": pbk,
            "serverName": q("sni"),
            "shortId": q("sid"),
            "spiderX": q("spx", "/") or "/",
        }
        pqv = q("pqv")
        if pqv:
            rs["mldsa65Verify"] = pqv
        stream_settings["realitySettings"] = rs
    elif security == "tls":
        ts: dict = {
            "serverName": q("sni") or host,
            "fingerprint": q("fp", "firefox") or "firefox",
        }
        alpn = q("alpn")
        if alpn:
            ts["alpn"] = [a.strip() for a in alpn.split(",") if a.strip()]
        if q("allowInsecure") in ("1", "true", "True"):
            ts["allowInsecure"] = True
        stream_settings["tlsSettings"] = ts

    if net_type == "ws":
        ws_cfg: dict = {"path": q("path", "/") or "/"}
        ws_host = q("host")
        if ws_host:
            ws_cfg["headers"] = {"Host": ws_host}
        stream_settings["wsSettings"] = ws_cfg
    elif net_type == "grpc":
        grpc_cfg: dict = {"serviceName": q("serviceName")}
        if q("mode") == "multi":
            grpc_cfg["multiMode"] = True
        stream_settings["grpcSettings"] = grpc_cfg
    elif net_type in ("xhttp", "splithttp"):
        stream_settings["network"] = "xhttp"
        xhttp_cfg: dict = {"path": q("path", "/") or "/"}
        if q("host"):
            xhttp_cfg["host"] = q("host")
        if q("mode"):
            xhttp_cfg["mode"] = q("mode")
        stream_settings["xhttpSettings"] = xhttp_cfg
    elif net_type == "httpupgrade":
        hu_cfg: dict = {"path": q("path", "/") or "/"}
        if q("host"):
            hu_cfg["host"] = q("host")
        stream_settings["httpupgradeSettings"] = hu_cfg

    settings: dict = {
        "address": host,
        "port": port,
        "id": uuid,
        "encryption": encryption,
    }
    if flow:
        settings["flow"] = flow

    outbound = {
        "tag": "isp-exit",
        "protocol": "vless",
        "settings": settings,
        "streamSettings": stream_settings,
    }

    return {
        "name": name,
        "host": host,
        "port": port,
        "security": security,
        "network": stream_settings["network"],
        "sni": q("sni") or "-",
        "flow": flow or "-",
        "outbound": outbound,
    }


def decode_subscription_body(raw: str) -> list[str]:
    text = raw.strip()
    if "vless://" in text or "vmess://" in text or "trojan://" in text or "hysteria" in text:
        return [line.strip() for line in text.splitlines() if line.strip()]
    # Try base64 (standard or urlsafe, ignoring whitespace)
    compact = "".join(text.split())
    pad = "=" * ((4 - len(compact) % 4) % 4)
    for decoder in (base64.b64decode, base64.urlsafe_b64decode):
        try:
            decoded = decoder(compact + pad).decode("utf-8", errors="ignore")
            if "://" in decoded:
                return [line.strip() for line in decoded.splitlines() if line.strip()]
        except Exception:
            continue
    return []


def fetch_subscription(url: str) -> tuple[list[dict], int]:
    """Fetch a subscription URL (trying direct, then local proxies) and parse VLESS profiles."""
    proxies_to_try: list[str | None] = [None]
    for port in (HTTP_PORT, 2080, 7897, 7890):
        with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
            s.settimeout(0.2)
            if s.connect_ex(("127.0.0.1", port)) == 0:
                proxies_to_try.append(f"http://127.0.0.1:{port}")

    last_err = ""
    body = ""
    for proxy in proxies_to_try:
        cmd = [CURL_BIN, "-fsSL", "--max-time", "15", "-A", "v2rayN/6.45"]
        if proxy:
            cmd.extend(["-x", proxy])
        else:
            cmd.extend(["--noproxy", "*"])
        cmd.append(url)
        res = subprocess.run(cmd, capture_output=True, text=True, check=False)
        if res.returncode == 0 and res.stdout.strip():
            body = res.stdout
            break
        last_err = res.stderr.strip() or f"curl exited {res.returncode}"

    if not body:
        raise RuntimeError(f"Failed to fetch subscription: {last_err}")

    lines = decode_subscription_body(body)
    profiles: list[dict] = []
    skipped = 0
    for line in lines:
        if line.startswith("vless://"):
            try:
                profiles.append(parse_vless_uri(line))
            except Exception:
                skipped += 1
        elif "://" in line:
            skipped += 1
    return profiles, skipped


def load_throne_profiles() -> list[dict]:
    if not THRONE_DB.is_file():
        return []
    conn = sqlite3.connect(f"file:{THRONE_DB}?mode=ro", uri=True)
    try:
        rows = conn.execute(
            "SELECT id, name, type, outbound_json FROM profiles WHERE type = 'xrayvless' ORDER BY gid, id"
        ).fetchall()
    finally:
        conn.close()

    profiles: list[dict] = []
    for _pid, name, _ptype, oj in rows:
        if not oj:
            continue
        try:
            ob = json.loads(oj)
        except json.JSONDecodeError:
            continue
        host, port = extract_host_port(ob)
        ss = ob.get("streamSettings") or {}
        rs = ss.get("realitySettings") or ss.get("tlsSettings") or {}
        profiles.append(
            {
                "name": name or f"{host}:{port}",
                "host": host,
                "port": port,
                "security": ss.get("security") or "none",
                "network": ss.get("network") or "raw",
                "sni": rs.get("serverName") or "-",
                "flow": (ob.get("settings") or {}).get("flow") or "-",
                "outbound": ob,
            }
        )
    return profiles


def load_saved_subs() -> list[dict[str, str]]:
    if not SUBS_PATH.is_file():
        return []
    try:
        data = json.loads(SUBS_PATH.read_text(encoding="utf-8"))
        if isinstance(data, list):
            return [s for s in data if isinstance(s, dict) and s.get("url")]
    except Exception:
        pass
    return []


def save_subs(subs: list[dict[str, str]]) -> None:
    write_secret_text(SUBS_PATH, json.dumps(subs, indent=2) + "\n")


def fzf_pick(items: list[str], header: str, prompt: str = "isp-ctl > ") -> int | None:
    if not items:
        return None
    numbered = [f"{i}\t{item}" for i, item in enumerate(items)]
    res = subprocess.run(
        [
            FZF_BIN,
            "--ansi",
            "--reverse",
            "--height=80%",
            "--border=rounded",
            "--delimiter=\t",
            "--with-nth=2..",
            "--no-sort",
            f"--prompt={prompt}",
            f"--header={header}",
        ],
        input="\n".join(numbered),
        capture_output=True,
        text=True,
        check=False,
    )
    if res.returncode != 0 or not res.stdout.strip():
        return None
    idx_str = res.stdout.split("\t", 1)[0].strip()
    return int(idx_str)


def probe_proxy_port(http_port: int) -> dict[str, str]:
    proxy_url = f"http://127.0.0.1:{http_port}"
    result = {"ip": "", "country": "", "city": "", "org": "", "timezone": "", "anthropic_ms": ""}

    for geo_url in ("https://ipinfo.io/json", "https://ipapi.co/json"):
        res = subprocess.run(
            [CURL_BIN, "-fsS", "--max-time", "6", "-x", proxy_url, geo_url],
            capture_output=True,
            text=True,
            check=False,
        )
        if res.returncode == 0 and res.stdout.strip():
            try:
                data = json.loads(res.stdout)
                ip = str(data.get("ip", "")).strip()
                if ip:
                    result["ip"] = ip
                    result["country"] = str(data.get("country") or data.get("country_code") or "")
                    result["city"] = str(data.get("city") or "")
                    result["org"] = str(data.get("org") or "")
                    result["timezone"] = str(data.get("timezone") or "")
                    break
            except json.JSONDecodeError:
                continue

    if not result["ip"]:
        res = subprocess.run(
            [CURL_BIN, "-fsS", "--max-time", "6", "-x", proxy_url, "https://api.ipify.org"],
            capture_output=True,
            text=True,
            check=False,
        )
        if res.returncode == 0:
            result["ip"] = res.stdout.strip()

    if result["ip"]:
        t0 = time.monotonic()
        res = subprocess.run(
            [CURL_BIN, "-sS", "--max-time", "6", "-o", "/dev/null", "-x", proxy_url, "https://api.anthropic.com/"],
            capture_output=True,
            text=True,
            check=False,
        )
        if res.returncode == 0:
            result["anthropic_ms"] = f"{int((time.monotonic() - t0) * 1000)}ms"

    return result


def validate_and_probe_candidate(outbound: dict) -> tuple[bool, str, dict[str, str]]:
    socks_port = find_free_port()
    http_port = find_free_port()
    cfg = build_xray_config(outbound, socks_port=socks_port, http_port=http_port)

    fd, tmp_path = tempfile.mkstemp(prefix="isp-ctl-test-", suffix=".json")
    proc: subprocess.Popen | None = None
    try:
        os.chmod(tmp_path, 0o600)
        with os.fdopen(fd, "w", encoding="utf-8") as f:
            json.dump(cfg, f, indent=2)

        chk = subprocess.run(
            [XRAY_BIN, "run", "-test", "-c", tmp_path],
            capture_output=True,
            text=True,
            check=False,
        )
        if chk.returncode != 0:
            err = (chk.stderr or chk.stdout).strip().splitlines()[-1:] or ["xray config invalid"]
            return False, f"Config validation failed: {err[0]}", {}

        proc = subprocess.Popen(
            [XRAY_BIN, "run", "-c", tmp_path],
            stdout=subprocess.DEVNULL,
            stderr=subprocess.PIPE,
            text=True,
        )
        # Wait up to 3s for temp HTTP port to bind
        ready = False
        for _ in range(30):
            if proc.poll() is not None:
                break
            with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
                if s.connect_ex(("127.0.0.1", http_port)) == 0:
                    ready = True
                    break
            time.sleep(0.1)

        if not ready:
            stderr_out = (proc.stderr.read() if proc.stderr else "").strip()
            return False, f"Temp xray failed to start: {stderr_out[:200]}", {}

        probe = probe_proxy_port(http_port)
        if not probe["ip"]:
            return False, "Connected to xray, but live exit-IP probe failed (proxy unreachable or blocked).", probe
        return True, "ok", probe
    finally:
        if proc is not None:
            proc.terminate()
            try:
                proc.wait(timeout=2)
            except subprocess.TimeoutExpired:
                proc.kill()
        if os.path.exists(tmp_path):
            os.unlink(tmp_path)


def prompt_yn(question: str, default: bool = False) -> bool:
    suffix = "[Y/n]" if default else "[y/N]"
    try:
        ans = input(f"{question} {suffix} ").strip().lower()
    except EOFError:
        return default
    if not ans:
        return default
    return ans in ("y", "yes")


def pause() -> None:
    try:
        input(f"\n{DIM}Press Enter to continue...{RESET}")
    except EOFError:
        pass


def apply_profile_flow(profile: dict) -> None:
    name = profile["name"]
    outbound = profile["outbound"]
    host, port = extract_host_port(outbound)

    print(f"\n{BOLD}Selected:{RESET} {CYAN}{name}{RESET}")
    print(f"  Server:    {host}:{port}")
    print(f"  Transport: {profile['network']} / {profile['security']} (SNI: {profile['sni']}, flow: {profile['flow']})")

    if host and not is_ip_literal(host):
        resolved = resolve_ipv4(host)
        if resolved:
            print(f"  Resolved:  {host} -> {resolved}")
            if prompt_yn(f"Pin server IPv4 ({resolved}) instead of hostname? (avoids DNS + lets Throne bypass RU)", default=True):
                outbound = set_outbound_host(outbound, resolved)

    print(f"\n{DIM}Running pre-flight xray validation & live exit-IP probe...{RESET}")
    ok, msg, probe = validate_and_probe_candidate(outbound)
    if not ok:
        print(f"{RED}Pre-flight failed:{RESET} {msg}")
        if not prompt_yn("Apply anyway (NOT recommended)?", default=False):
            return
    else:
        expected_ip = read_text(EXPECTED_IP_PATH)
        ip_match = bool(expected_ip and probe["ip"] == expected_ip)
        badge = f"{GREEN}MATCHES expected-ip{RESET}" if ip_match else f"{YELLOW}DIFFERS from expected-ip ({expected_ip or 'unset'}){RESET}"
        print(f"  Exit IP:   {BOLD}{probe['ip']}{RESET}  [{badge}]")
        loc = ", ".join(x for x in (probe["city"], probe["country"]) if x)
        if loc or probe["org"]:
            print(f"  Location:  {loc} ({probe['org']})")
        if probe["timezone"]:
            print(f"  Timezone:  {probe['timezone']}")
        if probe["anthropic_ms"]:
            print(f"  Anthropic: reachable ({probe['anthropic_ms']})")

    if not prompt_yn("\nActivate this profile in ~/.config/isp-proxy/config.json?", default=ok):
        print("Cancelled.")
        return

    ensure_cfg_dir()
    if CFG_PATH.is_file():
        shutil.copy2(CFG_PATH, BACKUP_PATH)
        os.chmod(BACKUP_PATH, 0o600)

    new_cfg = build_xray_config(outbound)
    write_secret_text(CFG_PATH, json.dumps(new_cfg, indent=2) + "\n")
    write_secret_text(ACTIVE_PROFILE_PATH, name + "\n")

    print(f"{DIM}Restarting isp-proxy.service...{RESET}")
    subprocess.run([SYSTEMCTL_BIN, "--user", "restart", "isp-proxy.service"], check=False)
    time.sleep(0.8)
    state = svc_active("isp-proxy.service", user=True)
    if state != "active":
        print(f"{RED}isp-proxy.service failed to start (state={state})! Rolling back...{RESET}")
        if BACKUP_PATH.is_file():
            shutil.copy2(BACKUP_PATH, CFG_PATH)
            os.chmod(CFG_PATH, 0o600)
            subprocess.run([SYSTEMCTL_BIN, "--user", "restart", "isp-proxy.service"], check=False)
        return

    print(f"{GREEN}Activated '{name}' — isp-proxy.service is active.{RESET}")

    live_ip = probe.get("ip", "")
    expected_ip = read_text(EXPECTED_IP_PATH)
    if live_ip and live_ip != expected_ip:
        if prompt_yn(f"Update ~/.config/isp-proxy/expected-ip to {live_ip}?", default=False):
            write_secret_text(EXPECTED_IP_PATH, live_ip + "\n")
            print(f"{GREEN}Updated expected-ip -> {live_ip}{RESET}")

    tz = probe.get("timezone", "")
    if tz and (Path(TZDATA_DIR) / tz).is_file():
        write_secret_text(TZ_PATH, tz + "\n")


def pick_from_profiles(profiles: list[dict], title: str) -> None:
    if not profiles:
        print(f"{YELLOW}No VLESS profiles found.{RESET}")
        pause()
        return

    items = [
        f"{p['name']:<34}  {DIM}{p['host']}:{p['port']}  [{p['network']}/{p['security']} · {p['sni']}]{RESET}"
        for p in profiles
    ]
    idx = fzf_pick(items, header=title, prompt="select profile > ")
    if idx is None:
        return
    apply_profile_flow(profiles[idx])
    pause()


def action_load_sub_url(url: str | None = None) -> None:
    if not url:
        try:
            url = input("\nSubscription URL (https://...): ").strip()
        except EOFError:
            return
    if not url:
        return

    print(f"{DIM}Fetching subscription...{RESET}")
    try:
        profiles, skipped = fetch_subscription(url)
    except Exception as e:
        print(f"{RED}{e}{RESET}")
        pause()
        return

    print(f"Found {GREEN}{len(profiles)}{RESET} VLESS profile(s)" + (f" ({skipped} non-VLESS skipped)." if skipped else "."))
    subs = load_saved_subs()
    if not any(s.get("url") == url for s in subs):
        if prompt_yn("Save this subscription URL locally for quick refresh later?", default=True):
            try:
                label = input("Label (e.g. ru-relay / aeza): ").strip() or urllib.parse.urlsplit(url).netloc or "sub"
            except EOFError:
                label = "sub"
            subs.append({"name": label, "url": url})
            save_subs(subs)

    pick_from_profiles(profiles, title=f"VLESS profiles from {url[:60]}")


def action_saved_subs() -> None:
    subs = load_saved_subs()
    if not subs:
        print(f"\n{YELLOW}No saved subscriptions yet. Use 'Load from Subscription URL' first.{RESET}")
        pause()
        return

    items = [f"{s.get('name', 'sub'):<20}  {DIM}{s['url'][:70]}{RESET}" for s in subs]
    items.append(f"{RED}Remove a saved subscription...{RESET}")
    idx = fzf_pick(items, header="Saved Subscriptions (stored in ~/.config/isp-proxy/subscriptions.json)", prompt="sub > ")
    if idx is None:
        return
    if idx == len(subs):
        rm_items = [f"{s.get('name', 'sub'):<20}  {s['url'][:70]}" for s in subs]
        rm_idx = fzf_pick(rm_items, header="Select subscription to remove", prompt="remove > ")
        if rm_idx is not None:
            removed = subs.pop(rm_idx)
            save_subs(subs)
            print(f"{YELLOW}Removed '{removed.get('name')}'.{RESET}")
            pause()
        return

    action_load_sub_url(subs[idx]["url"])


def action_paste_vless(uri: str | None = None) -> None:
    if not uri:
        try:
            uri = input("\nPaste vless:// URI: ").strip()
        except EOFError:
            return
    if not uri:
        return
    try:
        profile = parse_vless_uri(uri)
    except Exception as e:
        print(f"{RED}Invalid VLESS URI:{RESET} {e}")
        pause()
        return
    apply_profile_flow(profile)
    pause()


def action_import_throne() -> None:
    profiles = load_throne_profiles()
    if not profiles:
        print(f"\n{YELLOW}No VLESS profiles found in {THRONE_DB}.{RESET}")
        pause()
        return
    pick_from_profiles(profiles, title="Import VLESS profile from Throne DB (read-only)")


def action_health_check(interactive: bool = True) -> None:
    s = load_current_summary()
    print(f"\n{BOLD}=== ISP Proxy & Kill-Switch Health ==={RESET}")
    print(f"  Active profile:  {CYAN}{s['name']}{RESET} ({s['server']} · {s['security']} · SNI {s['sni']})")
    p_col = GREEN if s["proxy_state"] == "active" else RED
    f_col = GREEN if s["fw_state"] == "active" else RED
    print(f"  isp-proxy:       {p_col}{s['proxy_state']}{RESET}")
    print(f"  firewall unit:   {f_col}{s['fw_state']}{RESET}")
    print(f"  expected-ip:     {s['expected_ip']}")
    print(f"  cached TZ:       {s['tz']}")

    # Live kernel leak test inside isp-isolated.slice
    leak_res = subprocess.run(
        [
            "systemd-run",
            "--user",
            "--scope",
            "--quiet",
            "--collect",
            "--slice=isp-isolated.slice",
            "--",
            "timeout",
            "0.4",
            "bash",
            "-c",
            "exec 3<>/dev/tcp/1.1.1.1/443",
        ],
        capture_output=True,
        text=True,
        check=False,
    )
    if leak_res.returncode == 0:
        print(f"  kill-switch:     {RED}BREACH — direct 1.1.1.1:443 connected!{RESET}")
    else:
        print(f"  kill-switch:     {GREEN}BLOCKED (direct outbound dropped){RESET}")

    print(f"{DIM}Probing live exit via 127.0.0.1:{HTTP_PORT}...{RESET}")
    probe = probe_proxy_port(HTTP_PORT)
    if not probe["ip"]:
        print(f"  live exit:       {RED}FAILED (cannot reach internet via 127.0.0.1:{HTTP_PORT}){RESET}")
    else:
        match = probe["ip"] == s["expected_ip"]
        col = GREEN if match else YELLOW
        status = "MATCH" if match else "MISMATCH"
        print(f"  live exit IP:    {col}{probe['ip']} ({status}){RESET}")
        loc = ", ".join(x for x in (probe["city"], probe["country"]) if x)
        print(f"  location/ASN:    {loc} · {probe['org']}")
        print(f"  live timezone:   {probe['timezone']}")
        if probe["anthropic_ms"]:
            print(f"  Anthropic API:   {GREEN}reachable ({probe['anthropic_ms']}){RESET}")

    if interactive:
        pause()


def action_set_expected_ip() -> None:
    cur = read_text(EXPECTED_IP_PATH)
    print(f"\nCurrent expected-ip: {BOLD}{cur or '(unset)'}{RESET}")
    probe = probe_proxy_port(HTTP_PORT)
    live = probe.get("ip", "")
    if live:
        print(f"Current live exit IP: {CYAN}{live}{RESET} ({probe.get('country', '')} · {probe.get('org', '')})")
    try:
        entered = input(f"New expected IP (Enter to use live '{live}', or type IP): ").strip()
    except EOFError:
        return
    new_ip = entered or live
    if not new_ip or not is_ip_literal(new_ip):
        print(f"{RED}Invalid IP address: '{new_ip}'{RESET}")
        pause()
        return
    write_secret_text(EXPECTED_IP_PATH, new_ip + "\n")
    print(f"{GREEN}Updated ~/.config/isp-proxy/expected-ip -> {new_ip}{RESET}")
    pause()


def action_restart_logs() -> None:
    choice = fzf_pick(
        [
            "Restart isp-proxy.service",
            "View last 30 lines of isp-proxy logs",
            "Restart isp-isolated-anchor.service",
        ],
        header="Service Control",
        prompt="service > ",
    )
    if choice == 0:
        subprocess.run([SYSTEMCTL_BIN, "--user", "restart", "isp-proxy.service"], check=False)
        time.sleep(0.5)
        print(f"isp-proxy.service: {svc_active('isp-proxy.service', user=True)}")
        pause()
    elif choice == 1:
        subprocess.run([JOURNALCTL_BIN, "--user", "-u", "isp-proxy.service", "-n", "30", "--no-pager"], check=False)
        pause()
    elif choice == 2:
        subprocess.run([SYSTEMCTL_BIN, "--user", "restart", "isp-isolated-anchor.service"], check=False)
        print("Restarted anchor. If kill-switch says BREACH, run: sudo systemctl restart isp-killswitch-firewall")
        pause()


def interactive_loop() -> None:
    while True:
        s = load_current_summary()
        p_badge = f"{GREEN}●{RESET}" if s["proxy_state"] == "active" else f"{RED}●{RESET}"
        f_badge = f"{GREEN}●{RESET}" if s["fw_state"] == "active" else f"{RED}●{RESET}"
        header = (
            f"isp-proxy {p_badge} {s['proxy_state']}  |  kill-switch {f_badge} {s['fw_state']}\n"
            f"Active: {s['name']} ({s['server']} · {s['security']} · {s['sni']})\n"
            f"Expected IP: {s['expected_ip']}  |  TZ: {s['tz']}"
        )
        menu = [
            "1) Subscription URL     — fetch & pick VLESS profile from a sub link",
            "2) Saved subscriptions  — refresh or manage saved sub links",
            "3) Paste vless:// URI   — swap using a single VLESS link",
            "4) Import from Throne   — pick from Throne's VLESS profiles",
            "5) Live health check    — verify kill-switch, exit IP, ASN & TZ",
            "6) Set expected IP      — update ~/.config/isp-proxy/expected-ip",
            "7) Restart / Logs       — manage isp-proxy.service",
            "q) Quit",
        ]
        idx = fzf_pick(menu, header=header, prompt="isp-ctl > ")
        if idx is None or idx == 7:
            return
        if idx == 0:
            action_load_sub_url()
        elif idx == 1:
            action_saved_subs()
        elif idx == 2:
            action_paste_vless()
        elif idx == 3:
            action_import_throne()
        elif idx == 4:
            action_health_check(interactive=True)
        elif idx == 5:
            action_set_expected_ip()
        elif idx == 6:
            action_restart_logs()


def main(argv: list[str]) -> int:
    if len(argv) >= 2:
        cmd = argv[1]
        if cmd in ("-h", "--help", "help"):
            print("Usage: isp-ctl [sub <url> | uri <vless://...> | status]")
            return 0
        if cmd == "status":
            action_health_check(interactive=False)
            return 0
        if cmd == "sub":
            action_load_sub_url(argv[2] if len(argv) >= 3 else None)
            return 0
        if cmd == "uri":
            action_paste_vless(argv[2] if len(argv) >= 3 else None)
            return 0
    interactive_loop()
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv))
