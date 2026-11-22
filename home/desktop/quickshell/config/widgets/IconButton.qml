import QtQuick
import qs.common

Rectangle {
    id: root

    property alias icon: glyph.icon
    property alias iconSize: glyph.size
    property alias iconColor: glyph.color
    property alias fill: glyph.fill
    property alias hovered: state.containsMouse
    property int size: 32

    signal clicked(var mouse)

    implicitWidth: size
    implicitHeight: size
    radius: size / 2
    color: "transparent"

    Behavior on color {
        ColorAnimation { duration: Theme.durFast }
    }

    Icon {
        id: glyph
        anchors.centerIn: parent
        size: Math.round(root.size * 0.55)
    }

    StateLayer {
        id: state
        tint: glyph.color
        onClicked: mouse => root.clicked(mouse)
    }
}
