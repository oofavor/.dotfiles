pragma Singleton

import QtQuick
import Quickshell
import Quickshell.Bluetooth

Singleton {
    id: root

    readonly property BluetoothAdapter adapter: Bluetooth.defaultAdapter
    readonly property bool available: adapter !== null
    readonly property bool enabled: adapter?.enabled ?? false
    readonly property var devices: (adapter?.devices.values ?? []).filter(d => d.name || d.deviceName).slice().sort((a, b) => (b.connected - a.connected) || (b.paired - a.paired) || (a.name || "").localeCompare(b.name || ""))
    readonly property var connected: devices.filter(d => d.connected)

    readonly property string icon: !enabled ? "bluetooth_disabled" : connected.length > 0 ? "bluetooth_connected" : "bluetooth"
    readonly property string label: !available ? "Unavailable" : !enabled ? "Off" : connected.length === 1 ? connected[0].name : connected.length > 1 ? connected.length + " devices" : "On"

    function setEnabled(on) {
        if (adapter)
            adapter.enabled = on;
    }

    function setDiscovering(on) {
        if (adapter && adapter.enabled)
            adapter.discovering = on;
    }

    function deviceIcon(d) {
        const i = d.icon || "";
        if (i.includes("headset") || i.includes("headphone"))
            return "headphones";
        if (i.includes("audio"))
            return "speaker";
        if (i.includes("mouse"))
            return "mouse";
        if (i.includes("keyboard"))
            return "keyboard";
        if (i.includes("phone"))
            return "smartphone";
        if (i.includes("gaming") || i.includes("joystick"))
            return "stadia_controller";
        if (i.includes("computer"))
            return "computer";
        return "bluetooth";
    }
}
