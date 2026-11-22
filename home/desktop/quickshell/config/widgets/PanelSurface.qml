import QtQuick
import QtQuick.Effects
import qs.common

// Rounded panel background with a soft drop shadow, used by all popouts.
Item {
    id: root

    default property alias content: body.data
    property alias radius: bg.radius
    property alias color: bg.color

    RectangularShadow {
        anchors.fill: bg
        radius: bg.radius
        blur: 28
        spread: 0
        offset.y: 8
        color: Theme.shadow
    }

    Rectangle {
        id: bg
        anchors.fill: parent
        radius: Theme.radiusXl
        color: Theme.panelBg
        border.width: 1
        border.color: Theme.outline
    }

    // Swallow clicks so they don't fall through to the dismiss backdrop.
    MouseArea {
        anchors.fill: bg
        acceptedButtons: Qt.AllButtons
    }

    Item {
        id: body
        anchors.fill: parent
    }
}
