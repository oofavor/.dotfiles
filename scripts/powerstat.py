#!/usr/bin/env python3
"""Battery draw, per-process wakeups and interrupts over a time window.

A rootless stand-in for `powertop`: wakeups are counted from context switches
in /proc. Run it on battery with the machine otherwise idle (and Claude Code
closed, its spinner keeps the terminal redrawing).
"""

import os
import sys
import time

BAT = "/sys/class/power_supply/BAT0"


def read_procs():
    procs = {}
    for pid in filter(str.isdigit, os.listdir("/proc")):
        try:
            stat = open(f"/proc/{pid}/stat").read()
            fields = stat[stat.rindex(")") + 2 :].split()
            cpu = int(fields[11]) + int(fields[12])  # utime + stime, in jiffies
            ctx = 0
            for tid in os.listdir(f"/proc/{pid}/task"):
                for line in open(f"/proc/{pid}/task/{tid}/status"):
                    if "ctxt_switches" in line:
                        ctx += int(line.split()[1])
            cmd = open(f"/proc/{pid}/cmdline", "rb").read().replace(b"\0", b" ")
            name = cmd.decode(errors="replace")[:80] or f"[{stat[stat.index('(') + 1 : stat.rindex(')')]}]"
            procs[pid] = (ctx, cpu, name)
        except (OSError, ValueError, IndexError):
            pass  # process exited mid-read
    return procs


def read_irqs():
    irqs = {}
    for line in open("/proc/interrupts").readlines()[1:]:
        parts = line.split()
        count, i = 0, 1
        while i < len(parts) and parts[i].isdigit():
            count += int(parts[i])
            i += 1
        irqs[parts[0].rstrip(":")] = (count, " ".join(parts[i:])[-40:])
    return irqs


def main():
    seconds = float(sys.argv[1]) if len(sys.argv) > 1 else 30
    if open(f"{BAT}/status").read().strip() != "Discharging":
        print("note: not on battery, power_now is not meaningful\n")

    print(f"measuring for {seconds:.0f}s…", flush=True)
    procs0, irqs0, start = read_procs(), read_irqs(), time.time()
    watts = []
    while time.time() - start < seconds:
        time.sleep(2)
        watts.append(int(open(f"{BAT}/power_now").read()) / 1e6)
    procs1, irqs1, elapsed = read_procs(), read_irqs(), time.time() - start

    print(f"power: {sum(watts) / len(watts):.2f} W avg ({min(watts):.2f}–{max(watts):.2f}) over {elapsed:.0f}s")
    for light in os.scandir("/sys/class/backlight"):
        level = int(open(f"{light.path}/brightness").read())
        peak = int(open(f"{light.path}/max_brightness").read())
        print(f"brightness: {100 * level // peak}%")
    print()

    rows = [
        ((procs1[p][0] - procs0[p][0]) / elapsed, (procs1[p][1] - procs0[p][1]) / elapsed, procs1[p][2])
        for p in procs1
        if p in procs0
    ]
    rows.sort(reverse=True)
    print(" wakeups/s   cpu%  process")
    for wakeups, cpu, name in rows[:20]:
        print(f"{wakeups:9.1f} {cpu:6.1f}  {name}")
    print(f"    total {sum(r[0] for r in rows):.0f}/s\n")

    irq_rows = sorted(
        (((irqs1[k][0] - irqs0[k][0]) / elapsed), k, irqs1[k][1]) for k in irqs1 if k in irqs0
    )
    print("interrupts/s")
    for rate, key, name in reversed(irq_rows[-10:]):
        print(f"{rate:9.1f} {key:>6}  {name}")


if __name__ == "__main__":
    main()
