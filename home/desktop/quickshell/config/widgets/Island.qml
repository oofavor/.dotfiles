import QtQuick
import qs.common

// Floating bar "island": translucent rounded pill holding a row of content.
Rectangle {
    id: root

    default property alias content: row.data
    property alias spacing: row.spacing
    property int padding: 10
    property bool interactive: false
    property bool highlighted: false
    property alias tooltip: tip.text

    signal clicked(var mouse)
    signal scrolled(var wheel)

    implicitHeight: Theme.barHeight
    implicitWidth: row.implicitWidth + padding * 2
    radius: Theme.radiusMd
    color: highlighted ? Theme.alpha(Theme.surfaceHigh, 0.96) : Theme.islandBg
    border.width: 1
    border.color: highlighted ? Theme.outlineStrong : Theme.outline

    Behavior on implicitWidth {
        NumberAnimation { duration: Theme.durMed; easing.type: Easing.BezierSpline; easing.bezierCurve: Theme.curveDecel }
    }

    Behavior on color {
        ColorAnimation { duration: Theme.durFast }
    }

    StateLayer {
        visible: root.interactive
        radius: root.radius
        onClicked: mouse => root.clicked(mouse)
        onWheel: wheel => root.scrolled(wheel)
    }

    Row {
        id: row
        anchors.centerIn: parent
        spacing: 8
    }

    Tooltip {
        id: tip
    }
}
