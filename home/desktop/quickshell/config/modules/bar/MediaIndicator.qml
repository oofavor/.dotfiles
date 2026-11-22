import QtQuick
import qs.common
import qs.services
import qs.widgets

Island {
    id: root

    property string screenName: ""

    visible: Media.active
    spacing: 6
    padding: 10
    interactive: true
    onClicked: mouse => {
        if (mouse.button === Qt.LeftButton)
            Media.player.togglePlaying();
        else if (mouse.button === Qt.MiddleButton)
            Media.player.next();
        else
            Panels.toggle("dashboard", root.screenName);
    }

    Icon {
        anchors.verticalCenter: parent.verticalCenter
        icon: Media.playing ? "graphic_eq" : "pause"
        size: 16
        color: Theme.secondary
    }

    StyledText {
        anchors.verticalCenter: parent.verticalCenter
        width: Math.min(implicitWidth, 200)
        text: Media.title + (Media.artist ? "  ·  " + Media.artist : "")
        color: Theme.text
    }
}
