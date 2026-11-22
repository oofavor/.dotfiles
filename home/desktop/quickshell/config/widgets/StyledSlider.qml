import QtQuick
import QtQuick.Controls
import qs.common

// Thick pill slider with the icon living inside the filled track.
Slider {
    id: root

    property string icon: ""
    property string valueText: Math.round(value * 100) + "%"
    property color accent: Theme.accent

    signal iconClicked

    from: 0
    to: 1
    implicitHeight: 36
    padding: 0
    wheelEnabled: true
    stepSize: 0.01

    background: Rectangle {
        implicitWidth: 200
        implicitHeight: root.implicitHeight
        width: root.availableWidth
        height: root.availableHeight
        radius: height / 2
        color: Theme.surfaceHighest

        Rectangle {
            width: Math.max(height, root.visualPosition * parent.width)
            height: parent.height
            radius: height / 2
            color: root.accent

            Behavior on width {
                enabled: !root.pressed
                NumberAnimation { duration: Theme.durMed; easing.type: Easing.BezierSpline; easing.bezierCurve: Theme.curveDecel }
            }
        }

        Icon {
            anchors.left: parent.left
            anchors.leftMargin: 10
            anchors.verticalCenter: parent.verticalCenter
            icon: root.icon
            size: 18
            fill: true
            color: Theme.onAccent

            MouseArea {
                anchors.fill: parent
                anchors.margins: -8
                cursorShape: Qt.PointingHandCursor
                onClicked: root.iconClicked()
            }
        }

        StyledText {
            anchors.right: parent.right
            anchors.rightMargin: 14
            anchors.verticalCenter: parent.verticalCenter
            text: root.valueText
            font.pixelSize: Theme.fontSm
            font.features: ({ "tnum": 1 })
            color: root.visualPosition * parent.width > parent.width - 44 ? Theme.onAccent : Theme.textMuted
        }
    }

    handle: Item {}

    HoverHandler {
        cursorShape: Qt.PointingHandCursor
    }
}
