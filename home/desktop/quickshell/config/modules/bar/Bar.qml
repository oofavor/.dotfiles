import QtQuick
import Quickshell
import Quickshell.Wayland
import qs.common
import qs.services
import qs.widgets

PanelWindow {
    id: bar

    required property ShellScreen modelData

    screen: modelData
    anchors {
        top: true
        left: true
        right: true
    }
    implicitHeight: Theme.barTotal
    exclusiveZone: Theme.barTotal
    color: "transparent"

    WlrLayershell.namespace: "qs-bar"
    WlrLayershell.layer: WlrLayer.Top

    // Frosted glass behind each island (ext-background-effect, supported by niri).
    BackgroundEffect.blurRegion: Region {
        BlurRegion { target: logo; cornerRadius: Theme.radiusMd }
        BlurRegion { target: recording; cornerRadius: Theme.radiusMd }
        BlurRegion { target: workspaces; cornerRadius: Theme.radiusMd }
        BlurRegion { target: media; cornerRadius: Theme.radiusMd }
        BlurRegion { target: clock; cornerRadius: Theme.radiusMd }
        BlurRegion { target: tray; cornerRadius: Theme.radiusMd }
        BlurRegion { target: status; cornerRadius: Theme.radiusMd }
    }

    IdleInhibitor {
        window: bar
        enabled: Toggles.caffeine && bar.modelData === Quickshell.screens[0]
    }

    Item {
        anchors.fill: parent
        anchors.topMargin: Theme.barMargin
        anchors.leftMargin: Theme.barSide
        anchors.rightMargin: Theme.barSide

        // ── Left: launcher, workspaces, focused window ──
        Row {
            anchors.left: parent.left
            anchors.verticalCenter: parent.verticalCenter
            spacing: 6

            Island {
                id: logo
                padding: 0
                implicitWidth: Theme.barHeight
                interactive: true
                tooltip: "Launcher · right-click for overview"
                onClicked: mouse => {
                    if (mouse.button === Qt.RightButton)
                        Niri.toggleOverview();
                    else
                        Panels.toggle("launcher", bar.modelData.name);
                }

                StyledText {
                    width: Theme.barHeight
                    height: Theme.barHeight
                    horizontalAlignment: Text.AlignHCenter
                    text: "\uf313"
                    font.family: Theme.fontMono
                    font.pixelSize: 19
                    color: Theme.accent
                }
            }

            Workspaces {
                id: workspaces
                output: bar.modelData.name
            }

            ActiveWindow {
                output: bar.modelData.name
                maxWidth: Math.max(0, bar.width / 2 - centerRow.width / 2 - x - 40)
            }
        }

        // ── Center: media + clock ──
        Row {
            id: centerRow
            anchors.centerIn: parent
            spacing: 6

            MediaIndicator {
                id: media
                screenName: bar.modelData.name
            }

            ClockIsland {
                id: clock
                screenName: bar.modelData.name
            }
        }

        // ── Right: recording, tray, status ──
        Row {
            anchors.right: parent.right
            anchors.verticalCenter: parent.verticalCenter
            spacing: 6

            RecordingIndicator {
                id: recording
            }

            Tray {
                id: tray
                barWindow: bar
            }

            StatusIsland {
                id: status
                screenName: bar.modelData.name
            }
        }
    }
}
