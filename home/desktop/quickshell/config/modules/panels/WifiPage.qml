import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
import Quickshell
import qs.common
import qs.services
import qs.widgets

ColumnLayout {
    id: root

    signal back

    property var pending: null

    spacing: 8

    Component.onCompleted: Network.setScanning(true)
    Component.onDestruction: Network.setScanning(false)

    PageHeader {
        Layout.fillWidth: true
        title: "Wi-Fi"
        checked: Network.wifiEnabled
        busy: Network.wifiEnabled && Network.networks.some(n => n.stateChanging)
        onBack: root.back()
        onToggled: Network.setWifiEnabled(!Network.wifiEnabled)
    }

    ListView {
        id: list
        Layout.fillWidth: true
        Layout.preferredHeight: Math.min(contentHeight, 340)
        visible: Network.wifiEnabled
        clip: true
        spacing: 2
        model: Network.networks
        boundsBehavior: Flickable.StopAtBounds

        delegate: ListRow {
            id: row

            required property var modelData
            readonly property bool expanded: root.pending === modelData

            width: list.width
            icon: Network.signalIcon(modelData.signalStrength)
            title: modelData.name
            subtitle: modelData.stateChanging ? "Connecting…" : modelData.connected ? "Connected" : modelData.known ? "Saved" : ""
            trailingIcon: Network.isSecure(modelData) ? "lock" : ""
            highlighted: modelData.connected

            onClicked: {
                if (modelData.connected)
                    modelData.disconnect();
                else if (modelData.known || !Network.isSecure(modelData))
                    modelData.connect();
                else
                    root.pending = expanded ? null : modelData;
            }

            RowLayout {
                visible: row.expanded
                Layout.fillWidth: true
                spacing: 6

                TextField {
                    id: psk
                    Layout.fillWidth: true
                    placeholderText: "Password"
                    echoMode: TextInput.Password
                    color: Theme.text
                    placeholderTextColor: Theme.textDim
                    font.family: Theme.fontSans
                    font.pixelSize: Theme.fontMd
                    leftPadding: 12
                    onVisibleChanged: if (visible) forceActiveFocus()
                    background: Rectangle {
                        implicitHeight: 34
                        radius: Theme.radiusSm
                        color: Theme.surfaceHighest
                        border.width: psk.activeFocus ? 1 : 0
                        border.color: Theme.accent
                    }
                    onAccepted: {
                        row.modelData.connectWithPsk(text);
                        root.pending = null;
                    }
                }

                IconButton {
                    icon: "arrow_forward"
                    iconColor: Theme.accent
                    onClicked: psk.accepted()
                }
            }
        }
    }

    StyledText {
        Layout.fillWidth: true
        Layout.topMargin: 20
        Layout.bottomMargin: 20
        visible: !Network.wifiEnabled || Network.networks.length === 0
        horizontalAlignment: Text.AlignHCenter
        text: Network.wifiEnabled ? "Scanning for networks…" : "Wi-Fi is turned off"
        color: Theme.textMuted
    }

    ListRow {
        Layout.fillWidth: true
        icon: "settings"
        title: "Network settings"
        trailingIcon: "open_in_new"
        onClicked: {
            Panels.close();
            Quickshell.execDetached(["nm-connection-editor"]);
        }
    }
}
