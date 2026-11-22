# A place for Desktop Environment config

- `niri/` — compositor layout, window rules, keybinds, idle/lock.
- `quickshell/` — the desktop shell: bar, launcher, control center, dashboard
  (calendar + notification center), notifications, OSD, power menu, wallpaper.
  QML lives in `quickshell/config/`; colors, fonts and the wallpaper are
  injected from Stylix through the generated `common/Generated.qml`.

The palette ("Dusk") is defined once in `modules/desktop/stylix.nix`.

## Shell keybinds

| Keys            | Action                         |
| --------------- | ------------------------------ |
| `Mod+Space/R`   | launcher                       |
| `Mod+A`         | control center                 |
| `Mod+N` / `B`   | dashboard + notifications      |
| `Mod+Shift+N`   | toggle Do Not Disturb          |
| `Mod+Escape`    | power menu                     |
| `Mod+Alt+L`     | lock                           |
| `Mod+O`         | overview                       |
| `Mod+Shift+S`   | screenshot (region)            |
| `Mod+Shift+A`   | screenshot region → annotate   |
| `Mod+Shift+R`   | start/stop recording monitor   |
| `Mod+Shift+P`   | pick color (hex → clipboard)   |
| `Mod+Shift+T`   | OCR region → clipboard         |
| `Mod+Ctrl+V`    | clipboard history              |
| `Mod+Ctrl+.`    | emoji picker                   |
| `Mod+Shift+/`   | all keybinds                   |

IPC from scripts: `quickshell ipc call panels toggle <launcher|control|dashboard|power|wifi|bluetooth>`,
`quickshell ipc call panels launcher "<prefix>"` (launcher opened in a mode, e.g. `;` clipboard),
`quickshell ipc call notifications toggleDnd`, `quickshell ipc call toggles nightLight`,
`quickshell ipc call recorder <started|stopped>` (sent by `screen-record`; drives the bar timer).
