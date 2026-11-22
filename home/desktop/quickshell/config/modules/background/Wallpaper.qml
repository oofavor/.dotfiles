import QtQuick
import Quickshell
import Quickshell.Wayland
import qs.common

// Two layers per output: the sharp wallpaper inside workspaces, and a blurred,
// dimmed copy that niri places in the overview backdrop (see layer-rules).
Scope {
    id: root

    required property ShellScreen modelData

    PanelWindow {
        screen: root.modelData
        anchors {
            top: true
            bottom: true
            left: true
            right: true
        }
        exclusionMode: ExclusionMode.Ignore
        color: Theme.bg
        WlrLayershell.layer: WlrLayer.Background
        WlrLayershell.namespace: "qs-wallpaper"

        Image {
            anchors.fill: parent
            source: Generated.wallpaper
            fillMode: Image.PreserveAspectCrop
            asynchronous: true
            cache: false
            sourceSize.width: root.modelData.width * 2
        }
    }

    PanelWindow {
        screen: root.modelData
        anchors {
            top: true
            bottom: true
            left: true
            right: true
        }
        exclusionMode: ExclusionMode.Ignore
        color: Theme.bg
        WlrLayershell.layer: WlrLayer.Background
        WlrLayershell.namespace: "qs-backdrop"

        Image {
            anchors.fill: parent
            source: Generated.backdrop
            fillMode: Image.PreserveAspectCrop
            asynchronous: true
        }
    }
}
