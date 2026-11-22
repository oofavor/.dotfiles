pragma Singleton

import QtQuick
import Quickshell
import Quickshell.Io

// Which popout is open and on which output. Only one at a time.
Singleton {
    id: root

    // "", "control", "dashboard", "power", "launcher"
    property string current: ""
    property string output: ""
    // Control-center sub page: "main", "wifi", "bluetooth"
    property string controlPage: "main"
    // Text the launcher opens with, e.g. ";" for clipboard history.
    property string launcherPrefix: ""

    // "wifi" / "bluetooth" open the control center straight on that page.
    function toggle(name, output) {
        const page = name === "wifi" || name === "bluetooth" ? name : "main";
        const panel = page === "main" ? name : "control";
        if (current === panel && (panel !== "control" || controlPage === page)) {
            close();
            return;
        }
        open(name, output);
    }

    function open(name, output) {
        const page = name === "wifi" || name === "bluetooth" ? name : "main";
        root.output = output || Niri.focusedOutput;
        controlPage = page;
        current = page === "main" ? name : "control";
    }

    function close() {
        current = "";
    }

    function isOpenOn(name, screen) {
        if (current !== name)
            return false;
        const screens = Quickshell.screens;
        if (screens.some(s => s.name === output))
            return screen.name === output;
        return screen === screens[0];
    }

    IpcHandler {
        target: "panels"

        function toggle(name: string): void {
            root.toggle(name, "");
        }

        function close(): void {
            root.close();
        }

        // Launcher opened straight into a mode: `launcher ";"` is clipboard history.
        function launcher(prefix: string): void {
            if (root.current === "launcher") {
                root.close();
                return;
            }
            root.launcherPrefix = prefix;
            root.open("launcher", "");
        }
    }
}
