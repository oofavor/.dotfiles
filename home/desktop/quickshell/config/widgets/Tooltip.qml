import QtQuick
import Quickshell
import qs.common
import qs.services

// Hover hint for bar items. Drop inside the item it describes (not into a Row:
// it fills its parent); after a short delay a small popup appears below the bar.
Item {
    id: root

    property string text: ""
    property bool armed: false
    // Distance from this item's top to just below the bar window, set when armed.
    property real drop: height

    anchors.fill: parent

    HoverHandler {
        id: hover
        onHoveredChanged: if (!hovered)
            root.armed = false
    }

    Timer {
        interval: 450
        running: hover.hovered && !root.armed
        onTriggered: {
            const win = root.QsWindow.window;
            root.drop = win ? win.height - root.mapToItem(null, 0, 0).y + 6 : root.height + 6;
            root.armed = true;
        }
    }

    PopupWindow {
        visible: root.armed && root.text !== "" && Panels.current === ""
        anchor.item: root
        anchor.rect: Qt.rect(0, root.drop, root.width, 1)
        anchor.edges: Edges.Bottom
        anchor.gravity: Edges.Bottom
        implicitWidth: label.implicitWidth + 20
        implicitHeight: label.implicitHeight + 12
        color: "transparent"

        Rectangle {
            anchors.fill: parent
            radius: Theme.radiusSm
            color: Theme.toastBg
            border.width: 1
            border.color: Theme.outline

            StyledText {
                id: label
                anchors.centerIn: parent
                text: root.text
                font.pixelSize: Theme.fontSm
                horizontalAlignment: Text.AlignHCenter
                lineHeight: 1.15
            }
        }
    }
}
