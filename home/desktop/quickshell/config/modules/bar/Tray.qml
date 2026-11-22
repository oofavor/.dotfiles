import QtQuick
import Quickshell
import Quickshell.Services.SystemTray
import Quickshell.Widgets
import qs.common
import qs.widgets

Island {
    id: root

    property var barWindow

    visible: SystemTray.items.values.length > 0
    spacing: 4
    padding: 6

    Repeater {
        model: SystemTray.items

        Item {
            id: item

            required property SystemTrayItem modelData

            width: 24
            height: Theme.barHeight

            IconImage {
                anchors.centerIn: parent
                implicitSize: 16
                source: item.modelData.icon
            }

            Tooltip {
                text: item.modelData.tooltipTitle || item.modelData.title
            }

            MouseArea {
                anchors.fill: parent
                acceptedButtons: Qt.LeftButton | Qt.RightButton | Qt.MiddleButton
                cursorShape: Qt.PointingHandCursor
                onClicked: mouse => {
                    const m = item.modelData;
                    if (mouse.button === Qt.LeftButton && !m.onlyMenu)
                        m.activate();
                    else if (mouse.button === Qt.MiddleButton)
                        m.secondaryActivate();
                    else if (m.hasMenu) {
                        const p = item.mapToItem(null, 0, item.height + 6);
                        m.display(root.barWindow, p.x, p.y);
                    }
                }
            }
        }
    }
}
