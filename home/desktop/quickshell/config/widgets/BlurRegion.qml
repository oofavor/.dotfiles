import QtQuick
import Quickshell

// Blur region for a rounded, bordered item, pulled in under its 1px border.
// ext-background-effect regions are whole logical pixels with no antialiasing,
// so a region traced on the outer edge pokes out of the item's smooth corners
// as hard steps (2px ones at scale 2). Inset, the steps stay behind the fill.
//
// Geometry is computed instead of using `item:` because item mode only follows
// the item's own x/y, not its parents moving (e.g. the bar's centre row
// re-centring when the media island appears).
Region {
    id: root

    required property Item target
    property int cornerRadius: 0
    property int inset: 1

    readonly property bool shown: target !== null && target.visible
    readonly property real sceneX: offset(target, "x")
    readonly property real sceneY: offset(target, "y")

    // Window-relative position; reading every ancestor's x/y keeps it reactive.
    function offset(item, axis) {
        let sum = 0;
        for (let p = item; p !== null; p = p.parent)
            sum += p[axis];
        return sum;
    }

    x: shown ? Math.ceil(sceneX + inset) : 0
    y: shown ? Math.ceil(sceneY + inset) : 0
    width: shown ? Math.max(0, Math.floor(sceneX + target.width - inset) - x) : 0
    height: shown ? Math.max(0, Math.floor(sceneY + target.height - inset) - y) : 0
    radius: Math.max(0, cornerRadius - inset)
}
