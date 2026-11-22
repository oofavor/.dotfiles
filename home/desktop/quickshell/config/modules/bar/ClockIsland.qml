import QtQuick
import qs.common
import qs.services
import qs.widgets

Island {
    id: root

    property string screenName: ""

    spacing: 8
    interactive: true
    highlighted: Panels.current === "dashboard"
    onClicked: Panels.toggle("dashboard", root.screenName)
    tooltip: Time.format("dddd, d MMMM yyyy")

    StyledText {
        anchors.verticalCenter: parent.verticalCenter
        text: Time.format("ddd d MMM")
        color: Theme.textMuted
    }

    StyledText {
        anchors.verticalCenter: parent.verticalCenter
        text: Time.format("HH:mm")
        font.weight: Font.DemiBold
        font.features: ({ "tnum": 1 })
        color: Theme.textStrong
    }

    Row {
        anchors.verticalCenter: parent.verticalCenter
        visible: Notifs.dnd || Notifs.count > 0
        spacing: 3

        Icon {
            anchors.verticalCenter: parent.verticalCenter
            icon: Notifs.dnd ? "do_not_disturb_on" : "notifications_unread"
            size: 15
            fill: true
            color: Notifs.dnd ? Theme.textMuted : Theme.accent
        }

        StyledText {
            anchors.verticalCenter: parent.verticalCenter
            visible: Notifs.count > 0
            text: Notifs.count > 99 ? "99+" : Notifs.count
            font.pixelSize: Theme.fontSm
            font.weight: Font.Bold
            font.features: ({ "tnum": 1 })
            color: Notifs.dnd ? Theme.textMuted : Theme.accent
        }
    }
}
