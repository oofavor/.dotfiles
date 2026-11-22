import QtQuick
import QtQuick.Layouts
import qs.common
import qs.services
import qs.widgets

// Clock popout: date + calendar + media on the left, notification center on the right.
Popout {
    id: root

    width: 720
    height: 460

    onOpenChanged: if (open) calendar.reset()

    RowLayout {
        anchors.fill: parent
        anchors.margins: Theme.pad + 4
        spacing: 18

        ColumnLayout {
            Layout.preferredWidth: 300
            Layout.maximumWidth: 300
            Layout.fillWidth: false
            Layout.fillHeight: true
            spacing: 12

            ColumnLayout {
                Layout.leftMargin: 6
                spacing: -4

                StyledText {
                    text: Time.format("HH:mm")
                    font.pixelSize: Theme.fontHuge
                    font.weight: Font.Bold
                    font.features: ({ "tnum": 1 })
                    color: Theme.textStrong
                }

                StyledText {
                    text: Time.format("dddd, d MMMM")
                    font.pixelSize: Theme.fontLg
                    color: Theme.accent
                }
            }

            Card {
                Layout.fillWidth: true
                implicitHeight: calendar.implicitHeight + 20

                Calendar {
                    id: calendar
                    anchors.fill: parent
                    anchors.margins: 10
                }
            }

            Item {
                Layout.fillHeight: true
            }

            MediaCard {
                Layout.fillWidth: true
                visible: Media.player !== null
            }
        }

        Rectangle {
            Layout.fillHeight: true
            implicitWidth: 1
            color: Theme.outline
        }

        ColumnLayout {
            Layout.fillWidth: true
            Layout.fillHeight: true
            spacing: 10

            SectionHeader {
                Layout.fillWidth: true
                title: "Notifications"

                IconButton {
                    size: 30
                    icon: Notifs.dnd ? "notifications_off" : "notifications_active"
                    iconColor: Notifs.dnd ? Theme.accent : Theme.textMuted
                    onClicked: Notifs.dnd = !Notifs.dnd
                }

                IconButton {
                    size: 30
                    icon: "clear_all"
                    iconColor: Theme.textMuted
                    visible: Notifs.count > 0
                    onClicked: Notifs.clearAll()
                }
            }

            ListView {
                id: list
                Layout.fillWidth: true
                Layout.fillHeight: true
                visible: Notifs.count > 0
                clip: true
                spacing: 8
                model: Notifs.history
                boundsBehavior: Flickable.StopAtBounds

                delegate: NotificationCard {
                    required property var modelData
                    width: list.width
                    notification: modelData
                }

                add: Transition {
                    NumberAnimation { property: "opacity"; from: 0; to: 1; duration: Theme.durMed }
                }
            }

            ColumnLayout {
                Layout.fillWidth: true
                Layout.fillHeight: true
                visible: Notifs.count === 0
                spacing: 6

                Item {
                    Layout.fillHeight: true
                }

                Icon {
                    Layout.fillWidth: true
                    icon: Notifs.dnd ? "notifications_paused" : "done_all"
                    size: 40
                    color: Theme.textDim
                }

                StyledText {
                    Layout.fillWidth: true
                    horizontalAlignment: Text.AlignHCenter
                    text: Notifs.dnd ? "Do Not Disturb is on" : "You're all caught up"
                    color: Theme.textMuted
                }

                Item {
                    Layout.fillHeight: true
                }
            }
        }
    }
}
