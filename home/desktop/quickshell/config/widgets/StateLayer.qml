import QtQuick
import qs.common

// Hover / press feedback overlay + click handling. Drop inside any rounded item.
MouseArea {
    id: root

    property color tint: Theme.text
    property real radius: parent ? parent.radius ?? 0 : 0
    property bool disabled: false

    anchors.fill: parent
    hoverEnabled: true
    cursorShape: disabled ? Qt.ArrowCursor : Qt.PointingHandCursor
    acceptedButtons: Qt.LeftButton | Qt.RightButton | Qt.MiddleButton

    Rectangle {
        anchors.fill: parent
        radius: root.radius
        color: root.tint
        opacity: root.disabled ? 0 : root.pressed ? 0.14 : root.containsMouse ? 0.07 : 0

        Behavior on opacity {
            NumberAnimation { duration: Theme.durFast }
        }
    }
}
