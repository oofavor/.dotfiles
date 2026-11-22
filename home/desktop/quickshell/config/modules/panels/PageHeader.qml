import QtQuick
import QtQuick.Layouts
import qs.common
import qs.widgets

RowLayout {
    id: root

    property string title: ""
    property bool checked: false
    property bool busy: false

    signal back
    signal toggled

    spacing: 6

    IconButton {
        icon: "arrow_back"
        onClicked: root.back()
    }

    StyledText {
        Layout.fillWidth: true
        text: root.title
        font.pixelSize: Theme.fontLg
        font.weight: Font.DemiBold
        color: Theme.textStrong
    }

    Icon {
        visible: root.busy
        icon: "progress_activity"
        size: 18
        color: Theme.textMuted

        RotationAnimator on rotation {
            running: root.busy
            from: 0
            to: 360
            duration: 1000
            loops: Animation.Infinite
        }
    }

    // Compact switch
    Rectangle {
        implicitWidth: 40
        implicitHeight: 22
        radius: 11
        color: root.checked ? Theme.accent : Theme.surfaceHighest

        Behavior on color {
            ColorAnimation { duration: Theme.durFast }
        }

        Rectangle {
            x: root.checked ? parent.width - width - 3 : 3
            anchors.verticalCenter: parent.verticalCenter
            width: 16
            height: 16
            radius: 8
            color: root.checked ? Theme.onAccent : Theme.textMuted

            Behavior on x {
                NumberAnimation { duration: Theme.durMed; easing.type: Easing.BezierSpline; easing.bezierCurve: Theme.curveDecel }
            }
        }

        MouseArea {
            anchors.fill: parent
            cursorShape: Qt.PointingHandCursor
            onClicked: root.toggled()
        }
    }
}
