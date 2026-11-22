import QtQuick
import QtQuick.Layouts
import qs.common
import qs.services
import qs.widgets

Card {
    id: root

    implicitHeight: 84
    clip: true

    // Album art bleeding softly into the card background.
    Image {
        id: art
        anchors.fill: parent
        source: Media.artUrl
        fillMode: Image.PreserveAspectCrop
        opacity: status === Image.Ready ? 0.18 : 0
        asynchronous: true

        Behavior on opacity {
            NumberAnimation { duration: Theme.durSlow }
        }
    }

    RowLayout {
        anchors.fill: parent
        anchors.margins: 12
        spacing: 12

        Rectangle {
            Layout.preferredWidth: 60
            Layout.preferredHeight: 60
            radius: Theme.radiusMd
            color: Theme.surfaceHighest
            clip: true

            Icon {
                anchors.centerIn: parent
                visible: cover.status !== Image.Ready
                icon: "music_note"
                size: 26
                color: Theme.textMuted
            }

            Image {
                id: cover
                anchors.fill: parent
                source: Media.artUrl
                fillMode: Image.PreserveAspectCrop
                asynchronous: true
            }
        }

        ColumnLayout {
            Layout.fillWidth: true
            spacing: 2

            StyledText {
                Layout.fillWidth: true
                text: Media.title
                font.weight: Font.DemiBold
                color: Theme.textStrong
            }

            StyledText {
                Layout.fillWidth: true
                text: Media.artist
                font.pixelSize: Theme.fontSm
                color: Theme.textMuted
            }

            RowLayout {
                spacing: 2

                IconButton {
                    size: 28
                    icon: "skip_previous"
                    fill: true
                    onClicked: Media.player?.previous()
                }

                IconButton {
                    size: 30
                    icon: Media.playing ? "pause" : "play_arrow"
                    fill: true
                    color: Theme.accent
                    iconColor: Theme.onAccent
                    onClicked: Media.player?.togglePlaying()
                }

                IconButton {
                    size: 28
                    icon: "skip_next"
                    fill: true
                    onClicked: Media.player?.next()
                }
            }
        }
    }
}
