pragma Singleton

import QtQuick
import Quickshell
import Quickshell.Io

// Small system toggles that don't belong to a bigger service.
Singleton {
    id: root

    property bool nightLight: false
    property bool caffeine: false

    // wlsunset with a near-equal day/night range pins a constant warm temperature.
    Process {
        running: root.nightLight
        command: ["wlsunset", "-t", "3800", "-T", "3801"]
    }

    IpcHandler {
        target: "toggles"

        function nightLight(): void {
            root.nightLight = !root.nightLight;
        }

        function caffeine(): void {
            root.caffeine = !root.caffeine;
        }
    }
}
