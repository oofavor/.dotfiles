import QtQuick
import QtQuick.Layouts
import qs.common
import qs.widgets

// Generic two-line list row with leading icon and optional trailing icon.
Rectangle {
    id: root

    property string icon: ""
    property string title: ""
    property string subtitle: ""
    property string trailingIcon: ""
    property bool highlighted: false
    default property alias extra: extraCol.data

    signal clicked

    implicitHeight: Math.max(48, inner.implicitHeight + 16)
    radius: Theme.radiusMd
    color: highlighted ? Theme.accentSoft : "transparent"

    StateLayer {
        onClicked: root.clicked()
    }

    ColumnLayout {
        id: inner
        anchors.left: parent.left
        anchors.right: parent.right
        anchors.verticalCenter: parent.verticalCenter
        anchors.leftMargin: 10
        anchors.rightMargin: 10
        spacing: 6

        RowLayout {
            Layout.fillWidth: true
            spacing: 12

            Icon {
                icon: root.icon
                size: 20
                fill: root.highlighted
                color: root.highlighted ? Theme.accent : Theme.text
            }

            ColumnLayout {
                Layout.fillWidth: true
                spacing: 0

                StyledText {
                    Layout.fillWidth: true
                    text: root.title
                    color: root.highlighted ? Theme.textStrong : Theme.text
                    font.weight: root.highlighted ? Font.DemiBold : Font.Medium
                }

                StyledText {
                    Layout.fillWidth: true
                    visible: text !== ""
                    text: root.subtitle
                    font.pixelSize: Theme.fontSm
                    font.weight: Font.Normal
                    color: root.highlighted ? Theme.accent : Theme.textMuted
                }
            }

            Icon {
                visible: root.trailingIcon !== ""
                icon: root.trailingIcon
                size: 16
                color: Theme.textMuted
            }
        }

        ColumnLayout {
            id: extraCol
            Layout.fillWidth: true
        }
    }
}
