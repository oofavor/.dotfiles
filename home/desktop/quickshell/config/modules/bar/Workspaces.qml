import QtQuick
import qs.common
import qs.services
import qs.widgets

// Pill-style workspace indicator: active stretches into a capsule.
Island {
    id: root

    property string output: ""
    readonly property var list: Niri.workspacesOn(output)

    padding: 6
    spacing: 0

    WheelHandler {
        acceptedDevices: PointerDevice.Mouse | PointerDevice.TouchPad
        onWheel: event => {
            if (Math.abs(event.angleDelta.y) >= 60)
                Niri.focusWorkspaceRelative(event.angleDelta.y < 0 ? 1 : -1);
        }
    }

    Repeater {
        model: root.list

        Item {
            id: ws

            required property var modelData
            readonly property bool active: modelData.is_active
            readonly property bool occupied: Niri.windowCount(modelData.id) > 0

            width: dot.width + 10
            height: Theme.barHeight

            Rectangle {
                id: dot
                anchors.centerIn: parent
                width: ws.active ? 22 : 8
                height: 8
                radius: 4
                color: ws.modelData.is_urgent ? Theme.red : ws.active ? Theme.accent : ws.occupied ? Theme.alpha(Theme.text, 0.55) : Theme.alpha(Theme.text, 0.18)

                Behavior on width {
                    NumberAnimation { duration: Theme.durMed; easing.type: Easing.BezierSpline; easing.bezierCurve: Theme.curveDecel }
                }
                Behavior on color {
                    ColorAnimation { duration: Theme.durMed }
                }
            }

            MouseArea {
                anchors.fill: parent
                cursorShape: Qt.PointingHandCursor
                onClicked: Niri.focusWorkspace(ws.modelData.id)
            }
        }
    }
}
