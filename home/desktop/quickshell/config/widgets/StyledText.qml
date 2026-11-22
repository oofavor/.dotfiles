import QtQuick
import qs.common

Text {
    color: Theme.text
    font.family: Theme.fontSans
    font.pixelSize: Theme.fontMd
    font.weight: Font.Medium
    verticalAlignment: Text.AlignVCenter
    elide: Text.ElideRight
    renderType: Text.NativeRendering

    Behavior on color {
        ColorAnimation { duration: Theme.durFast }
    }
}
