import QtQuick
import QtQuick.Layouts
import Quickshell
import Quickshell.Io
import qs.common
import qs.services
import qs.widgets

Popout {
    id: root

    readonly property string page: Panels.controlPage

    width: 380
    height: column.implicitHeight + Theme.pad * 2

    onOpenChanged: {
        if (open) {
            uptime.reload();
            Brightness.refresh();
        }
    }

    Behavior on height {
        NumberAnimation { duration: Theme.durMed; easing.type: Easing.BezierSpline; easing.bezierCurve: Theme.curveDecel }
    }

    FileView {
        id: uptime
        path: "/proc/uptime"
    }

    ColumnLayout {
        id: column
        anchors.fill: parent
        anchors.margins: Theme.pad
        spacing: 12

        // ── Header ──
        RowLayout {
            Layout.fillWidth: true
            spacing: 10

            Rectangle {
                Layout.preferredWidth: 40
                Layout.preferredHeight: 40
                radius: 20
                gradient: Gradient {
                    orientation: Gradient.Vertical
                    GradientStop { position: 0; color: Theme.secondary }
                    GradientStop { position: 1; color: Theme.accent }
                }

                StyledText {
                    anchors.centerIn: parent
                    text: (Quickshell.env("USER") || "?").charAt(0).toUpperCase()
                    font.pixelSize: Theme.fontXl
                    font.weight: Font.Bold
                    color: Theme.onAccent
                }
            }

            ColumnLayout {
                Layout.fillWidth: true
                spacing: 0

                StyledText {
                    Layout.fillWidth: true
                    text: Quickshell.env("USER") || "user"
                    font.pixelSize: Theme.fontLg
                    font.weight: Font.DemiBold
                    color: Theme.textStrong
                }

                StyledText {
                    Layout.fillWidth: true
                    text: {
                        const s = parseFloat(uptime.text().split(" ")[0]) || 0;
                        const d = Math.floor(s / 86400), h = Math.floor(s % 86400 / 3600), m = Math.floor(s % 3600 / 60);
                        return "up " + (d > 0 ? d + "d " : "") + (h > 0 ? h + "h " : "") + m + "m";
                    }
                    font.pixelSize: Theme.fontSm
                    color: Theme.textMuted
                }
            }

            IconButton {
                icon: "lock"
                onClicked: {
                    Panels.close();
                    Quickshell.execDetached(Generated.lockCommand);
                }
            }

            IconButton {
                icon: "power_settings_new"
                iconColor: hovered ? Theme.red : Theme.text
                onClicked: Panels.open("power", Panels.output)
            }
        }

        // ── Pages ──
        Loader {
            Layout.fillWidth: true
            sourceComponent: root.page === "wifi" ? wifiPage : root.page === "bluetooth" ? btPage : mainPage
        }
    }

    Component {
        id: mainPage

        ColumnLayout {
            spacing: 12

            GridLayout {
                Layout.fillWidth: true
                columns: 2
                rowSpacing: 8
                columnSpacing: 8

                QuickTile {
                    Layout.fillWidth: true
                    icon: Network.wiredConnected ? "lan" : Network.wifiEnabled ? "wifi" : "wifi_off"
                    label: Network.wiredConnected ? "Ethernet" : "Wi-Fi"
                    sublabel: Network.label
                    active: Network.wifiEnabled || Network.wiredConnected
                    hasDetails: true
                    onToggled: Network.setWifiEnabled(!Network.wifiEnabled)
                    onDetailsRequested: Panels.controlPage = "wifi"
                }

                QuickTile {
                    Layout.fillWidth: true
                    icon: BluetoothService.icon
                    label: "Bluetooth"
                    sublabel: BluetoothService.label
                    active: BluetoothService.enabled
                    hasDetails: BluetoothService.available
                    onToggled: BluetoothService.setEnabled(!BluetoothService.enabled)
                    onDetailsRequested: Panels.controlPage = "bluetooth"
                }

                QuickTile {
                    Layout.fillWidth: true
                    icon: Notifs.dnd ? "do_not_disturb_on" : "notifications"
                    label: "Do Not Disturb"
                    sublabel: Notifs.dnd ? "On" : "Off"
                    active: Notifs.dnd
                    onToggled: Notifs.dnd = !Notifs.dnd
                }

                QuickTile {
                    Layout.fillWidth: true
                    icon: "nightlight"
                    label: "Night Light"
                    sublabel: Toggles.nightLight ? "Warm" : "Off"
                    active: Toggles.nightLight
                    onToggled: Toggles.nightLight = !Toggles.nightLight
                }

                QuickTile {
                    Layout.fillWidth: true
                    icon: "coffee"
                    label: "Keep Awake"
                    sublabel: Toggles.caffeine ? "Idle blocked" : "Off"
                    active: Toggles.caffeine
                    onToggled: Toggles.caffeine = !Toggles.caffeine
                }

                QuickTile {
                    Layout.fillWidth: true
                    icon: Audio.micMuted ? "mic_off" : "mic"
                    label: "Microphone"
                    sublabel: Audio.micMuted ? "Muted" : "On"
                    active: !Audio.micMuted
                    onToggled: Audio.toggleMicMute()
                }
            }

            Card {
                Layout.fillWidth: true
                implicitHeight: sliders.implicitHeight + 24

                ColumnLayout {
                    id: sliders
                    anchors.fill: parent
                    anchors.margins: 12
                    spacing: 10

                    RowLayout {
                        Layout.fillWidth: true
                        spacing: 8

                        StyledSlider {
                            Layout.fillWidth: true
                            icon: Audio.icon
                            value: Audio.volume
                            accent: Audio.muted ? Theme.textDim : Theme.accent
                            onMoved: Audio.setVolume(value)
                            onIconClicked: Audio.toggleMute()
                        }

                        IconButton {
                            icon: "tune"
                            iconColor: Theme.textMuted
                            onClicked: {
                                Panels.close();
                                Quickshell.execDetached(["pwvucontrol"]);
                            }
                        }
                    }

                    RowLayout {
                        Layout.fillWidth: true
                        visible: Brightness.available
                        spacing: 8

                        StyledSlider {
                            Layout.fillWidth: true
                            icon: "light_mode"
                            value: Brightness.value
                            accent: Theme.yellow
                            onMoved: Brightness.set(value)
                        }

                        // Keeps both sliders the same length.
                        Item {
                            implicitWidth: 32
                        }
                    }
                }
            }

            Card {
                Layout.fillWidth: true
                visible: Display.available
                implicitHeight: 56

                RowLayout {
                    anchors.fill: parent
                    anchors.leftMargin: 14
                    anchors.rightMargin: 10
                    spacing: 12

                    Icon {
                        icon: "speed"
                        size: 22
                    }

                    ColumnLayout {
                        Layout.fillWidth: true
                        spacing: 0

                        StyledText {
                            Layout.fillWidth: true
                            text: "Refresh rate"
                            font.weight: Font.DemiBold
                            color: Theme.textStrong
                        }

                        StyledText {
                            Layout.fillWidth: true
                            text: Display.hz(Display.activeRate) + " Hz" + (Display.choice === 0 ? " · auto" : "")
                            font.pixelSize: Theme.fontSm
                            color: Theme.textMuted
                        }
                    }

                    // Segmented picker: Auto, then each rate the panel offers.
                    Rectangle {
                        implicitWidth: segments.implicitWidth + 6
                        implicitHeight: 32
                        radius: Theme.radiusMd
                        color: Theme.surfaceHighest

                        Row {
                            id: segments
                            anchors.centerIn: parent
                            spacing: 2

                            Repeater {
                                model: [0, ...Display.rates]

                                Rectangle {
                                    id: segment

                                    required property int modelData
                                    readonly property bool selected: Display.choice === modelData

                                    width: Math.max(36, segmentLabel.implicitWidth + 16)
                                    height: 26
                                    radius: Theme.radiusSm
                                    color: selected ? Theme.accent : "transparent"

                                    Behavior on color {
                                        ColorAnimation { duration: Theme.durFast }
                                    }

                                    StateLayer {
                                        tint: segment.selected ? Theme.onAccent : Theme.text
                                        onClicked: Display.setChoice(segment.modelData)
                                    }

                                    StyledText {
                                        id: segmentLabel
                                        anchors.centerIn: parent
                                        text: segment.modelData === 0 ? "Auto" : Display.hz(segment.modelData)
                                        font.pixelSize: Theme.fontSm
                                        font.weight: segment.selected ? Font.DemiBold : Font.Medium
                                        font.features: ({ "tnum": 1 })
                                        color: segment.selected ? Theme.onAccent : Theme.text
                                    }
                                }
                            }
                        }
                    }
                }
            }

            Card {
                Layout.fillWidth: true
                visible: Battery.available
                implicitHeight: 56

                RowLayout {
                    anchors.fill: parent
                    anchors.leftMargin: 14
                    anchors.rightMargin: 14
                    spacing: 12

                    Icon {
                        icon: Battery.icon
                        size: 22
                        fill: true
                                    color: Battery.charging ? Theme.green : Battery.low ? Theme.yellow : Theme.text
                    }

                    ColumnLayout {
                        Layout.fillWidth: true
                        spacing: 0

                        StyledText {
                            text: Math.round(Battery.percent * 100) + "%"
                            font.weight: Font.DemiBold
                            color: Theme.textStrong
                        }

                        StyledText {
                            Layout.fillWidth: true
                            text: Battery.status
                            font.pixelSize: Theme.fontSm
                            color: Theme.textMuted
                        }
                    }

                    StyledText {
                        visible: Battery.watts > 0.1
                        text: Battery.watts.toFixed(1) + " W"
                        font.pixelSize: Theme.fontSm
                        font.features: ({ "tnum": 1 })
                        color: Theme.textMuted
                    }
                }
            }
        }
    }

    Component {
        id: wifiPage

        WifiPage {
            onBack: Panels.controlPage = "main"
        }
    }

    Component {
        id: btPage

        BluetoothPage {
            onBack: Panels.controlPage = "main"
        }
    }
}
