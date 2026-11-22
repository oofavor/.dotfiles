import QtQuick
import Quickshell
import Quickshell.Widgets
import qs.common
import qs.services
import qs.widgets

// App icon + title of the active window on this output's active workspace.
Item {
    id: root

    property string output: ""
    property real maxWidth: 400

    readonly property var workspace: Niri.workspacesOn(output).find(w => w.is_active) ?? null
    readonly property var window: workspace?.active_window_id != null ? Niri.windows[workspace.active_window_id] ?? null : null
    readonly property var entry: window ? DesktopEntries.heuristicLookup(window.app_id) : null

    implicitWidth: window ? Math.min(maxWidth, row.implicitWidth + 12) : 0
    implicitHeight: Theme.barHeight
    clip: true
    opacity: window ? 1 : 0

    Behavior on opacity {
        NumberAnimation { duration: Theme.durMed }
    }

    Row {
        id: row
        anchors.left: parent.left
        anchors.leftMargin: 6
        anchors.verticalCenter: parent.verticalCenter
        spacing: 8

        IconImage {
            anchors.verticalCenter: parent.verticalCenter
            implicitSize: 16
            source: Quickshell.iconPath(root.entry?.icon ?? root.window?.app_id ?? "", "application-x-executable")
        }

        StyledText {
            anchors.verticalCenter: parent.verticalCenter
            width: Math.min(implicitWidth, root.maxWidth - 40)
            text: root.window?.title || root.entry?.name || root.window?.app_id || ""
            color: Theme.textMuted
        }
    }
}
