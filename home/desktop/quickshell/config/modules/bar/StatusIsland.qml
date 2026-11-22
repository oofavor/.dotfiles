import QtQuick
import qs.common
import qs.services
import qs.widgets

// Compact system status; click opens the control center.
Island {
    id: root

    property string screenName: ""

    spacing: 10
    interactive: true
    highlighted: Panels.current === "control"
    onClicked: Panels.toggle("control", root.screenName)
    onScrolled: wheel => Audio.setVolume(Audio.volume + (wheel.angleDelta.y > 0 ? 0.05 : -0.05))

    // ── Privacy: something is capturing right now ──
    Icon {
        anchors.verticalCenter: parent.verticalCenter
        visible: Privacy.screen
        icon: "screen_share"
        size: 16
        fill: true
        color: Theme.accent

        Tooltip {
            text: "Screen shared with " + (Privacy.apps(Privacy.screenLinks) || "an app")
        }
    }

    Icon {
        anchors.verticalCenter: parent.verticalCenter
        visible: Privacy.camera
        icon: "videocam"
        size: 16
        fill: true
        color: Theme.accent

        Tooltip {
            text: "Camera in use by " + (Privacy.apps(Privacy.cameraLinks) || "an app")
        }
    }

    Icon {
        anchors.verticalCenter: parent.verticalCenter
        visible: Privacy.mic
        icon: Audio.micMuted ? "mic_off" : "mic"
        size: 16
        fill: true
        color: Audio.micMuted ? Theme.textDim : Theme.accent

        Tooltip {
            text: "Microphone in use by " + (Privacy.apps(Privacy.micLinks) || "an app") + (Audio.micMuted ? " (muted)" : "")
        }
    }

    StyledText {
        anchors.verticalCenter: parent.verticalCenter
        visible: Niri.keyboardLayouts.length > 1
        text: Niri.keyboardLayout
        font.pixelSize: Theme.fontSm
        font.weight: Font.Bold
        color: Theme.textMuted

        Tooltip {
            text: Niri.keyboardLayouts[Niri.keyboardLayoutIdx] ?? ""
        }
    }

    Icon {
        anchors.verticalCenter: parent.verticalCenter
        visible: Toggles.caffeine
        icon: "coffee"
        size: 16
        fill: true
        color: Theme.yellow

        Tooltip {
            text: "Keep awake is on"
        }
    }

    Icon {
        anchors.verticalCenter: parent.verticalCenter
        icon: Network.icon
        size: 16
        fill: true

        Tooltip {
            text: Network.active && !Network.wiredConnected ? Network.label + " · " + Math.round(Network.strength * 100) + "%" : Network.label
        }
    }

    Icon {
        anchors.verticalCenter: parent.verticalCenter
        visible: BluetoothService.enabled
        icon: BluetoothService.icon
        size: 16

        Tooltip {
            text: "Bluetooth · " + BluetoothService.label
        }
    }

    Icon {
        anchors.verticalCenter: parent.verticalCenter
        icon: Audio.icon
        size: 16
        fill: true
        color: Audio.muted ? Theme.textDim : Theme.text

        Tooltip {
            text: Audio.deviceName + " · " + (Audio.muted ? "muted" : Math.round(Audio.volume * 100) + "%")
        }
    }

    Item {
        anchors.verticalCenter: parent.verticalCenter
        visible: Battery.available
        implicitWidth: battery.implicitWidth
        implicitHeight: battery.implicitHeight

        Row {
            id: battery
            spacing: 3

            Icon {
                anchors.verticalCenter: parent.verticalCenter
                icon: Battery.icon
                size: 16
                fill: true
                color: Battery.charging ? Theme.green : Battery.critical ? Theme.red : Battery.low ? Theme.yellow : Theme.text
            }

            StyledText {
                anchors.verticalCenter: parent.verticalCenter
                text: Math.round(Battery.percent * 100) + "%"
                font.features: ({ "tnum": 1 })
            }
        }

        Tooltip {
            text: Battery.status + (Battery.watts > 0.1 ? " · " + Battery.watts.toFixed(1) + " W" : "")
        }
    }
}
