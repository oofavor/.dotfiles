pragma Singleton

import QtQuick
import Quickshell
import Quickshell.Networking

Singleton {
    id: root

    readonly property var devices: Networking.devices.values
    readonly property var wifiDevice: devices.find(d => d.type === DeviceType.Wifi) ?? null
    readonly property var wiredDevice: devices.find(d => d.type === DeviceType.Wired) ?? null

    readonly property bool wifiEnabled: Networking.wifiEnabled
    readonly property bool wiredConnected: wiredDevice?.connected ?? false
    readonly property var networks: (wifiDevice?.networks.values ?? []).slice().sort((a, b) => (b.connected - a.connected) || (b.known - a.known) || (b.signalStrength - a.signalStrength))
    readonly property var active: networks.find(n => n.connected) ?? null
    readonly property real strength: active?.signalStrength ?? 0

    readonly property string label: wiredConnected ? "Ethernet" : active ? active.name : wifiEnabled ? "Not connected" : "Off"

    readonly property string icon: {
        if (wiredConnected)
            return "lan";
        if (!wifiEnabled)
            return "signal_wifi_off";
        if (!active)
            return "signal_wifi_statusbar_not_connected";
        return signalIcon(strength);
    }

    function signalIcon(s) {
        if (s > 0.75)
            return "signal_wifi_4_bar";
        if (s > 0.5)
            return "network_wifi_3_bar";
        if (s > 0.25)
            return "network_wifi_2_bar";
        return "network_wifi_1_bar";
    }

    function isSecure(network) {
        return network.security !== WifiSecurityType.Open && network.security !== WifiSecurityType.Unknown;
    }

    function setWifiEnabled(on) {
        Networking.wifiEnabled = on;
    }

    function setScanning(on) {
        if (wifiDevice)
            wifiDevice.scannerEnabled = on;
    }
}
