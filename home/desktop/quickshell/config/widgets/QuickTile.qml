import QtQuick
import QtQuick.Layouts
import qs.common

// Control-center toggle tile. Body toggles; optional chevron opens a detail page.
Rectangle {
    id: root

    property string icon: ""
    property string label: ""
    property string sublabel: ""
    property bool active: false
    property bool hasDetails: false

    signal toggled
    signal detailsRequested

    implicitHeight: 56
    radius: Theme.radiusLg
    color: active ? Theme.accent : Theme.surfaceHigh
    border.width: active ? 0 : 1
    border.color: Theme.outline

    Behavior on color {
        ColorAnimation { duration: Theme.durMed }
    }

    StateLayer {
        tint: root.active ? Theme.onAccent : Theme.text
        onClicked: root.toggled()
    }

    RowLayout {
        anchors.fill: parent
        anchors.leftMargin: 12
        anchors.rightMargin: root.hasDetails ? 4 : 12
        spacing: 10

        Rectangle {
            Layout.preferredWidth: 32
            Layout.preferredHeight: 32
            radius: 16
            color: root.active ? Theme.alpha(Theme.onAccent, 0.12) : Theme.surfaceHighest

            Icon {
                anchors.centerIn: parent
                icon: root.icon
                size: 18
                fill: root.active
                color: root.active ? Theme.onAccent : Theme.text
            }
        }

        ColumnLayout {
            Layout.fillWidth: true
            spacing: 0

            StyledText {
                Layout.fillWidth: true
                text: root.label
                font.weight: Font.DemiBold
                color: root.active ? Theme.onAccent : Theme.text
            }

            StyledText {
                Layout.fillWidth: true
                visible: text !== ""
                text: root.sublabel
                font.pixelSize: Theme.fontSm
                font.weight: Font.Normal
                color: root.active ? Theme.alpha(Theme.onAccent, 0.75) : Theme.textMuted
            }
        }

        IconButton {
            visible: root.hasDetails
            size: 30
            icon: "chevron_right"
            iconColor: root.active ? Theme.onAccent : Theme.textMuted
            onClicked: root.detailsRequested()
        }
    }
}
