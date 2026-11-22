import QtQuick
import Quickshell
import qs.services

// Keeps the built-in panel on the rate picked in services/Display.qml. niri
// resets modes to its config on every reload, so re-apply after those too.
Scope {
    id: root

    function apply() {
        if (Display.mode !== "")
            Quickshell.execDetached(["niri", "msg", "output", Display.output, "mode", Display.mode]);
    }

    Connections {
        target: Display

        function onModeChanged() {
            root.apply();
        }
    }

    Connections {
        target: Niri

        function onConfigLoaded() {
            root.apply();
        }
    }

    Component.onCompleted: apply()
}
