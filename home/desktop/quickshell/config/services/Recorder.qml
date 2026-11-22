pragma Singleton

import QtQuick
import Quickshell
import Quickshell.Io

// State of the screen-record script (home/desktop/screen-tools.nix). The script
// reports start/stop over IPC; the pidfile check covers a shell restart mid-recording.
Singleton {
    id: root

    property bool active: false
    property real startedAt: 0
    property int elapsed: 0

    readonly property string pidfile: (Quickshell.env("XDG_RUNTIME_DIR") || "/tmp") + "/screen-record.pid"
    readonly property string elapsedText: {
        const h = Math.floor(elapsed / 3600), m = Math.floor(elapsed % 3600 / 60), s = elapsed % 60;
        const pad = n => (n < 10 ? "0" : "") + n;
        return (h > 0 ? h + ":" + pad(m) : m) + ":" + pad(s);
    }

    // SIGINT lets gpu-screen-recorder finalise the file; the script then reports back.
    function stop() {
        Quickshell.execDetached(["sh", "-c", 'kill -INT "$(cat "$1")"', "sh", pidfile]);
    }

    Timer {
        running: root.active
        repeat: true
        interval: 1000
        triggeredOnStart: true
        onTriggered: root.elapsed = Math.max(0, Math.floor((Date.now() - root.startedAt) / 1000))
    }

    Process {
        running: true
        command: ["sh", "-c", 'kill -0 "$(cat "$1")" 2>/dev/null && stat -c %Y "$1"', "sh", root.pidfile]
        stdout: StdioCollector {
            onStreamFinished: {
                const t = parseInt(text);
                if (t > 0) {
                    root.startedAt = t * 1000;
                    root.active = true;
                }
            }
        }
    }

    IpcHandler {
        target: "recorder"

        function started(): void {
            root.startedAt = Date.now();
            root.active = true;
        }

        function stopped(): void {
            root.active = false;
        }
    }
}
