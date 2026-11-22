//@ pragma UseQApplication
//@ pragma Env QT_QUICK_CONTROLS_STYLE=Basic

import QtQuick
import Quickshell
import qs.modules.background
import qs.modules.bar
import qs.modules.lock
import qs.modules.panels
import qs.modules.notifications
import qs.modules.osd
import qs.modules.power

ShellRoot {
    Variants {
        model: Quickshell.screens

        Scope {
            id: perScreen

            required property ShellScreen modelData

            Wallpaper {
                modelData: perScreen.modelData
            }

            Bar {
                modelData: perScreen.modelData
            }

            PanelHost {
                modelData: perScreen.modelData
            }
        }
    }

    LockScreen {}
    NotificationPopups {}
    Osd {}
    RefreshRate {}
    BatteryBrightness {}
}
