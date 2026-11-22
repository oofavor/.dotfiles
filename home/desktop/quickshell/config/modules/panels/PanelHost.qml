import QtQuick
import Quickshell
import Quickshell.Wayland
import qs.common
import qs.services

// One overlay surface per output that hosts every popout. It only maps while
// something is open (plus the closing animation) and grabs the keyboard so
// Escape works; clicking anywhere outside a panel closes it.
PanelWindow {
    id: host

    required property ShellScreen modelData

    readonly property bool controlOpen: Panels.isOpenOn("control", modelData)
    readonly property bool dashboardOpen: Panels.isOpenOn("dashboard", modelData)
    readonly property bool powerOpen: Panels.isOpenOn("power", modelData)
    readonly property bool launcherOpen: Panels.isOpenOn("launcher", modelData)
    readonly property bool anyOpen: controlOpen || dashboardOpen || powerOpen || launcherOpen

    screen: modelData
    visible: anyOpen || lingering.running
    anchors {
        top: true
        bottom: true
        left: true
        right: true
    }
    exclusionMode: ExclusionMode.Ignore
    color: "transparent"

    WlrLayershell.layer: WlrLayer.Overlay
    WlrLayershell.namespace: "qs-panels"
    WlrLayershell.keyboardFocus: anyOpen ? WlrKeyboardFocus.Exclusive : WlrKeyboardFocus.None

    BackgroundEffect.blurRegion: Region {
        PopoutRegion { popout: control }
        PopoutRegion { popout: dashboard }
        PopoutRegion { popout: launcher }
        Region { item: host.powerOpen ? backdrop : null }
    }

    onAnyOpenChanged: if (!anyOpen) lingering.restart()

    Timer {
        id: lingering
        interval: Theme.durMed + 60
    }

    FocusScope {
        anchors.fill: parent
        focus: true

        Keys.onEscapePressed: Panels.close()

        Rectangle {
            id: backdrop
            anchors.fill: parent
            color: host.powerOpen ? Theme.alpha("#000000", 0.55) : "transparent"

            Behavior on color {
                ColorAnimation { duration: Theme.durMed }
            }

            MouseArea {
                anchors.fill: parent
                acceptedButtons: Qt.AllButtons
                onClicked: Panels.close()
            }
        }

        ControlCenter {
            id: control
            anchors.top: parent.top
            anchors.right: parent.right
            anchors.topMargin: Theme.barTotal + Theme.gap
            anchors.rightMargin: Theme.barSide
            open: host.controlOpen
        }

        Dashboard {
            id: dashboard
            anchors.top: parent.top
            anchors.horizontalCenter: parent.horizontalCenter
            anchors.topMargin: Theme.barTotal + Theme.gap
            open: host.dashboardOpen
        }

        Launcher {
            id: launcher
            anchors.top: parent.top
            anchors.horizontalCenter: parent.horizontalCenter
            anchors.topMargin: Math.round(parent.height * 0.2)
            open: host.launcherOpen
        }

        PowerMenu {
            anchors.centerIn: parent
            open: host.powerOpen
        }
    }
}
