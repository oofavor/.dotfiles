pragma Singleton

import QtQuick
import Quickshell
import Quickshell.Io

// Backlight via brightnessctl. Keybinds go through IPC so the OSD always reacts.
Singleton {
    id: root

    property real value: 0
    readonly property bool available: max > 0
    property int max: 0
    // Ceiling to apply once the next read lands; -1 when none is pending.
    property real pendingCap: -1

    signal changedByUser

    function refresh() {
        reader.running = true;
    }

    function write(v) {
        const pct = Math.round(Math.max(0.01, Math.min(1, v)) * 100);
        value = pct / 100;
        writer.command = ["brightnessctl", "-q", "set", pct + "%"];
        writer.running = true;
    }

    function set(v) {
        write(v);
        changedByUser();
    }

    // Lower brightness to at most v, never raise it. Re-reads first because
    // swayidle and the keys change the backlight behind this service's back.
    function cap(v) {
        pendingCap = v;
        refresh();
    }

    function step(delta) {
        set(value + delta);
    }

    Process {
        id: reader
        running: true
        command: ["brightnessctl", "-m", "info"]
        stdout: StdioCollector {
            onStreamFinished: {
                // name,class,current,percent,max
                const parts = text.trim().split(",");
                if (parts.length >= 5) {
                    root.max = parseInt(parts[4]);
                    root.value = parseInt(parts[2]) / root.max;
                }
                if (root.pendingCap >= 0) {
                    if (root.value > root.pendingCap)
                        root.write(root.pendingCap);
                    root.pendingCap = -1;
                }
            }
        }
    }

    Process {
        id: writer
    }

    IpcHandler {
        target: "brightness"

        function increment(): void {
            root.step(0.05);
        }

        function decrement(): void {
            root.step(-0.05);
        }

        function set(percent: int): void {
            root.set(percent / 100);
        }
    }
}
