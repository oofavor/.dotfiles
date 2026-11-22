import QtQuick
import qs.common

// Material Symbols Rounded glyph, referenced by ligature name ("wifi", "bolt", ...).
Text {
    id: root

    property string icon: ""
    property real size: 16
    property bool fill: false
    property int weight: 400

    text: icon
    color: Theme.text
    font.family: Theme.fontIcon
    font.pixelSize: size
    font.variableAxes: ({
            "FILL": fill ? 1 : 0,
            "wght": weight,
            "opsz": Math.max(20, Math.min(48, size))
        })
    horizontalAlignment: Text.AlignHCenter
    verticalAlignment: Text.AlignVCenter
    renderType: Text.NativeRendering

    Behavior on color {
        ColorAnimation { duration: Theme.durFast }
    }
}
