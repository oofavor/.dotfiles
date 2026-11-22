import QtQuick
import QtQuick.Layouts
import Quickshell
import Quickshell.Bluetooth
import qs.common
import qs.services
import qs.widgets

ColumnLayout {
    id: root

    signal back

    spacing: 8

    Component.onCompleted: BluetoothService.setDiscovering(true)
    Component.onDestruction: BluetoothService.setDiscovering(false)

    PageHeader {
        Layout.fillWidth: true
        title: "Bluetooth"
        checked: BluetoothService.enabled
        busy: BluetoothService.adapter?.discovering ?? false
        onBack: root.back()
        onToggled: BluetoothService.setEnabled(!BluetoothService.enabled)
    }

    ListView {
        id: list
        Layout.fillWidth: true
        Layout.preferredHeight: Math.min(contentHeight, 340)
        visible: BluetoothService.enabled
        clip: true
        spacing: 2
        model: BluetoothService.devices
        boundsBehavior: Flickable.StopAtBounds

        delegate: ListRow {
            required property var modelData

            width: list.width
            icon: BluetoothService.deviceIcon(modelData)
            title: modelData.name || modelData.deviceName
            subtitle: {
                const d = modelData;
                if (d.pairing)
                    return "Pairing…";
                if (d.state === BluetoothDeviceState.Connecting)
                    return "Connecting…";
                if (d.connected)
                    return d.batteryAvailable ? "Connected · " + Math.round(d.battery * 100) + "%" : "Connected";
                return d.paired ? "Paired" : "";
            }
            highlighted: modelData.connected

            onClicked: {
                const d = modelData;
                if (d.connected)
                    d.disconnect();
                else if (d.paired)
                    d.connect();
                else {
                    d.trusted = true;
                    d.pair();
                }
            }
        }
    }

    StyledText {
        Layout.fillWidth: true
        Layout.topMargin: 20
        Layout.bottomMargin: 20
        visible: !BluetoothService.enabled || BluetoothService.devices.length === 0
        horizontalAlignment: Text.AlignHCenter
        text: BluetoothService.enabled ? "Looking for devices…" : "Bluetooth is turned off"
        color: Theme.textMuted
    }

    ListRow {
        Layout.fillWidth: true
        icon: "settings"
        title: "Bluetooth settings"
        trailingIcon: "open_in_new"
        onClicked: {
            Panels.close();
            Quickshell.execDetached(["blueman-manager"]);
        }
    }
}
