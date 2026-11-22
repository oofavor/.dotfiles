import QtQuick
import Quickshell
import Quickshell.Services.UPower
import qs.common
import qs.services

// Like macOS's "slightly dim the display on battery": on unplug (and at login on
// battery) brightness drops to the cap if it was above it. Raising it again
// afterwards sticks until the next unplug.
Scope {
    id: root

    function apply() {
        if (UPower.onBattery)
            Brightness.cap(Generated.batteryBrightnessCap);
    }

    Connections {
        target: UPower

        function onOnBatteryChanged() {
            root.apply();
        }
    }

    Component.onCompleted: apply()
}
