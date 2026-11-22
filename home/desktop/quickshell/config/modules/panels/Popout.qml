import QtQuick
import qs.common
import qs.widgets

// Panel surface that fades + slides in from the bar when `open` flips.
PanelSurface {
    id: root

    property bool open: false
    // How far it slides in from; 0 for a plain fade.
    property real slideDistance: 10
    // Current slide-in offset, so PopoutRegion can follow it.
    readonly property real slide: slideIn.y

    visible: opacity > 0
    opacity: open ? 1 : 0
    transform: Translate {
        id: slideIn
        y: root.open ? 0 : -root.slideDistance

        Behavior on y {
            NumberAnimation { duration: Theme.durMed; easing.type: Easing.BezierSpline; easing.bezierCurve: Theme.curveDecel }
        }
    }

    Behavior on opacity {
        NumberAnimation { duration: Theme.durMed; easing.type: Easing.BezierSpline; easing.bezierCurve: Theme.curveStandard }
    }
}
