import QtQuick
import QtQuick.Layouts
import qs.common

RowLayout {
    id: root

    property string title: ""
    default property alias actions: actionRow.data

    spacing: 8

    StyledText {
        Layout.fillWidth: true
        text: root.title
        font.pixelSize: Theme.fontLg
        font.weight: Font.DemiBold
        color: Theme.textStrong
    }

    Row {
        id: actionRow
        spacing: 4
    }
}
