import QtQuick
import Quickshell
import Quickshell.Wayland
import qs.common
import qs.services
import qs.widgets

// Volume / brightness indicator near the bottom edge of the focused output.
PanelWindow {
    id: root

    property string kind: "volume"
    property bool shown: false
    property bool armed: false

    readonly property real value: kind === "volume" ? Audio.volume : Brightness.value
    readonly property string icon: kind === "volume" ? Audio.icon : "light_mode"
    readonly property bool dimmed: kind === "volume" && Audio.muted

    function show(k) {
        if (!armed)
            return;
        kind = k;
        shown = true;
        hideTimer.restart();
    }

    screen: Quickshell.screens.find(s => s.name === Niri.focusedOutput) ?? Quickshell.screens[0]
    visible: shown || pill.opacity > 0
    anchors.bottom: true
    margins.bottom: 56
    exclusionMode: ExclusionMode.Ignore
    implicitWidth: 280
    implicitHeight: 52
    color: "transparent"
    mask: Region {}

    WlrLayershell.layer: WlrLayer.Overlay
    WlrLayershell.namespace: "qs-osd"

    // Explicit geometry rather than `item: pill`: item regions don't follow the
    // pill's scale animation, so the blur would stay at the 0.92 starting size.
    // Inset 1px under the border like BlurRegion, so the region's pixel steps
    // stay inside the rounded ends.
    BackgroundEffect.blurRegion: Region {
        x: 1
        y: 1
        width: root.shown ? root.implicitWidth - 2 : 0
        height: root.shown ? root.implicitHeight - 2 : 0
        radius: root.implicitHeight / 2 - 1
    }

    // Ignore the burst of change signals while services settle at startup.
    Timer {
        running: true
        interval: 1500
        onTriggered: root.armed = true
    }

    Timer {
        id: hideTimer
        interval: 1400
        onTriggered: root.shown = false
    }

    Connections {
        target: Audio.sink?.audio ?? null

        function onVolumeChanged() {
            root.show("volume");
        }

        function onMutedChanged() {
            root.show("volume");
        }
    }

    Connections {
        target: Brightness

        function onChangedByUser() {
            root.show("brightness");
        }
    }

    PanelSurface {
        id: pill
        anchors.fill: parent
        radius: height / 2
        opacity: root.shown ? 1 : 0
        scale: root.shown ? 1 : 0.92

        Behavior on opacity {
            NumberAnimation { duration: Theme.durMed }
        }

        Behavior on scale {
            NumberAnimation { duration: Theme.durMed; easing.type: Easing.BezierSpline; easing.bezierCurve: Theme.curveDecel }
        }

        Row {
            anchors.centerIn: parent
            spacing: 14

            Icon {
                anchors.verticalCenter: parent.verticalCenter
                icon: root.icon
                size: 20
                fill: true
                color: root.dimmed ? Theme.textDim : Theme.accent
            }

            Rectangle {
                anchors.verticalCenter: parent.verticalCenter
                width: 170
                height: 6
                radius: 3
                color: Theme.surfaceHighest

                Rectangle {
                    width: parent.width * Math.min(1, root.value)
                    height: parent.height
                    radius: 3
                    color: root.dimmed ? Theme.textDim : Theme.accent

                    Behavior on width {
                        NumberAnimation { duration: Theme.durFast }
                    }
                }
            }

            StyledText {
                anchors.verticalCenter: parent.verticalCenter
                width: 34
                text: Math.round(root.value * 100)
                horizontalAlignment: Text.AlignRight
                font.features: ({ "tnum": 1 })
                color: Theme.textStrong
            }
        }
    }
}
