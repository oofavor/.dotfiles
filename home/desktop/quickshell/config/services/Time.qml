pragma Singleton

import QtQuick
import Quickshell

Singleton {
    readonly property date now: clock.date

    function format(fmt) {
        return Qt.formatDateTime(clock.date, fmt);
    }

    SystemClock {
        id: clock
        // Nothing shows seconds; minute ticks save a wakeup every second.
        precision: SystemClock.Minutes
    }
}
