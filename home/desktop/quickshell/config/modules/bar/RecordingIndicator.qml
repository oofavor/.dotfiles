import QtQuick
import qs.common
import qs.services
import qs.widgets

// Pulsing dot + elapsed time while screen-record runs; click stops it.
Island {
    id: root

    visible: Recorder.active
    spacing: 7
    interactive: true
    tooltip: "Recording screen · click to stop"
    onClicked: Recorder.stop()

    Rectangle {
        anchors.verticalCenter: parent.verticalCenter
        width: 8
        height: 8
        radius: 4
        color: Theme.red

        SequentialAnimation on opacity {
            running: Recorder.active
            loops: Animation.Infinite
            NumberAnimation { to: 0.3; duration: 900; easing.type: Easing.InOutSine }
            NumberAnimation { to: 1; duration: 900; easing.type: Easing.InOutSine }
        }
    }

    StyledText {
        anchors.verticalCenter: parent.verticalCenter
        text: Recorder.elapsedText
        font.weight: Font.DemiBold
        font.features: ({ "tnum": 1 })
        color: Theme.textStrong
    }
}
