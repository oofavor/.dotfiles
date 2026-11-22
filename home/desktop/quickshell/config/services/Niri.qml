pragma Singleton

import QtQuick
import Quickshell
import Quickshell.Io

// Live niri state from the IPC event stream, plus a request socket for actions.
Singleton {
    id: root

    property var workspaces: []
    property var windows: ({})
    property var focusedWindowId: null
    property var keyboardLayouts: []
    property int keyboardLayoutIdx: 0
    property bool overviewOpen: false

    signal configLoaded

    readonly property var focusedWindow: focusedWindowId !== null ? (windows[focusedWindowId] ?? null) : null
    readonly property string focusedOutput: workspaces.find(w => w.is_focused)?.output ?? ""
    readonly property string keyboardLayout: shortLayout(keyboardLayouts[keyboardLayoutIdx] ?? "")

    function shortLayout(name) {
        const known = {
            "English (US)": "EN",
            "Russian": "RU"
        };
        return known[name] ?? name.slice(0, 2).toUpperCase();
    }

    function workspacesOn(output) {
        return workspaces.filter(w => w.output === output);
    }

    function windowCount(workspaceId) {
        let n = 0;
        for (const id in windows)
            if (windows[id].workspace_id === workspaceId)
                n++;
        return n;
    }

    function send(request) {
        requests.write(JSON.stringify(request) + "\n");
        requests.flush();
    }

    function action(name, args) {
        send({
            "Action": {
                [name]: args ?? {}
            }
        });
    }

    // Launch through niri so the process lives in its own scope instead of
    // quickshell.service's cgroup, which systemd kills when the shell restarts.
    function spawn(command) {
        action("Spawn", {
            "command": command
        });
    }

    function focusWorkspace(id) {
        action("FocusWorkspace", {
            "reference": {
                "Id": id
            }
        });
    }

    function focusWorkspaceRelative(delta) {
        action(delta > 0 ? "FocusWorkspaceDown" : "FocusWorkspaceUp");
    }

    function switchLayout() {
        action("SwitchLayout", {
            "layout": "Next"
        });
    }

    function toggleOverview() {
        action("ToggleOverview");
    }

    function quit() {
        action("Quit", {
            "skip_confirmation": true
        });
    }

    function sortWorkspaces(list) {
        return list.slice().sort((a, b) => a.output === b.output ? a.idx - b.idx : a.output.localeCompare(b.output));
    }

    function handle(event) {
        if (event.WorkspacesChanged) {
            workspaces = sortWorkspaces(event.WorkspacesChanged.workspaces);
        } else if (event.WorkspaceActivated) {
            const {
                id,
                focused
            } = event.WorkspaceActivated;
            const output = workspaces.find(w => w.id === id)?.output;
            workspaces = workspaces.map(w => Object.assign({}, w, {
                    "is_active": w.output === output ? w.id === id : w.is_active,
                    "is_focused": focused ? w.id === id : w.is_focused
                }));
        } else if (event.WorkspaceActiveWindowChanged) {
            const e = event.WorkspaceActiveWindowChanged;
            workspaces = workspaces.map(w => w.id === e.workspace_id ? Object.assign({}, w, {
                    "active_window_id": e.active_window_id
                }) : w);
        } else if (event.WorkspaceUrgencyChanged) {
            const e = event.WorkspaceUrgencyChanged;
            workspaces = workspaces.map(w => w.id === e.id ? Object.assign({}, w, {
                    "is_urgent": e.urgent
                }) : w);
        } else if (event.WindowsChanged) {
            const map = {};
            for (const w of event.WindowsChanged.windows) {
                map[w.id] = w;
                if (w.is_focused)
                    focusedWindowId = w.id;
            }
            windows = map;
        } else if (event.WindowOpenedOrChanged) {
            const w = event.WindowOpenedOrChanged.window;
            const map = Object.assign({}, windows);
            map[w.id] = w;
            windows = map;
            if (w.is_focused)
                focusedWindowId = w.id;
        } else if (event.WindowClosed) {
            const map = Object.assign({}, windows);
            delete map[event.WindowClosed.id];
            windows = map;
            if (focusedWindowId === event.WindowClosed.id)
                focusedWindowId = null;
        } else if (event.WindowFocusChanged) {
            focusedWindowId = event.WindowFocusChanged.id;
        } else if (event.KeyboardLayoutsChanged) {
            keyboardLayouts = event.KeyboardLayoutsChanged.keyboard_layouts.names;
            keyboardLayoutIdx = event.KeyboardLayoutsChanged.keyboard_layouts.current_idx;
        } else if (event.KeyboardLayoutSwitched) {
            keyboardLayoutIdx = event.KeyboardLayoutSwitched.idx;
        } else if (event.OverviewOpenedOrClosed) {
            overviewOpen = event.OverviewOpenedOrClosed.is_open;
        } else if (event.ConfigLoaded && !event.ConfigLoaded.failed) {
            configLoaded();
        }
    }

    Process {
        id: stream
        running: true
        command: ["niri", "msg", "--json", "event-stream"]
        stdout: SplitParser {
            onRead: line => {
                try {
                    root.handle(JSON.parse(line));
                } catch (e) {
                    console.warn("niri: bad event", e);
                }
            }
        }
        // Reconnect if niri restarts or the stream dies.
        onExited: restart.start()
    }

    Timer {
        id: restart
        interval: 1000
        onTriggered: stream.running = true
    }

    Socket {
        id: requests
        path: Quickshell.env("NIRI_SOCKET")
        connected: true
        onConnectedChanged: if (!connected) reconnect.start()
        parser: SplitParser {
            onRead: line => {
                if (!line.includes("\"Ok\""))
                    console.warn("niri:", line);
            }
        }
    }

    Timer {
        id: reconnect
        interval: 1000
        onTriggered: requests.connected = true
    }
}
