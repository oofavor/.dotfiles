import QtQuick
import Quickshell
import qs.common

// Blur region that tracks a Popout through its slide-in. Region's `item` mode
// only re-maps when the item's own geometry changes, not its transform, so it
// would freeze the blur at the slide's starting offset. Coordinates are
// window-relative; popouts sit in PanelHost's full-window FocusScope.
// Inset 1px under the border like BlurRegion, so the region's pixel steps
// stay inside the rounded corners.
Region {
    required property Popout popout

    x: Math.ceil(popout.x + 1)
    y: Math.ceil(popout.y + popout.slide + 1)
    width: popout.open ? Math.floor(popout.x + popout.width - 1) - x : 0
    height: popout.open ? Math.floor(popout.y + popout.slide + popout.height - 1) - y : 0
    radius: Theme.radiusXl - 1
}
