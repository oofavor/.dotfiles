import QtQuick
import Quickshell
import Quickshell.Wayland
import qs.common
import qs.services
import qs.widgets

// Toasts stacked under the bar on the focused output.
PanelWindow {
    id: root

    screen: Quickshell.screens.find(s => s.name === Niri.focusedOutput) ?? Quickshell.screens[0]
    visible: Notifs.popups.length > 0 && Panels.current !== "dashboard"
    anchors {
        top: true
        right: true
    }
    margins {
        top: Theme.barTotal + Theme.gap
        right: Theme.barSide
    }
    exclusionMode: ExclusionMode.Ignore
    implicitWidth: 380
    implicitHeight: Math.max(1, stack.implicitHeight + 40)
    color: "transparent"

    WlrLayershell.layer: WlrLayer.Overlay
    WlrLayershell.namespace: "qs-notifications"

    // Only the cards take input; the shadow margin stays click-through.
    mask: Region {
        item: stack
    }

    Column {
        id: stack
        width: parent.width
        spacing: Theme.gap

        Repeater {
            model: Notifs.popups

            Item {
                id: wrapper

                required property var modelData

                width: stack.width
                height: card.implicitHeight

                function hide() {
                    exit.start();
                }

                Timer {
                    running: !card.hovered && !card.critical
                    interval: {
                        const t = wrapper.modelData.expireTimeout;
                        return t > 0 ? Math.max(3000, Math.min(15000, t * 1000)) : 5500;
                    }
                    onTriggered: wrapper.hide()
                }

                SequentialAnimation {
                    id: exit
                    ParallelAnimation {
                        NumberAnimation { target: card; property: "x"; to: 60; duration: Theme.durMed; easing.type: Easing.BezierSpline; easing.bezierCurve: Theme.curveAccel }
                        NumberAnimation { target: card; property: "opacity"; to: 0; duration: Theme.durMed }
                    }
                    ScriptAction {
                        script: Notifs.removePopup(wrapper.modelData)
                    }
                }

                NotificationCard {
                    id: card
                    width: parent.width
                    notification: wrapper.modelData
                    popup: true
                    opacity: 0
                    x: 60

                    Component.onCompleted: enter.start()

                    ParallelAnimation {
                        id: enter
                        NumberAnimation { target: card; property: "x"; to: 0; duration: Theme.durSlow; easing.type: Easing.BezierSpline; easing.bezierCurve: Theme.curveDecel }
                        NumberAnimation { target: card; property: "opacity"; to: 1; duration: Theme.durMed }
                    }
                }
            }
        }
    }
}
