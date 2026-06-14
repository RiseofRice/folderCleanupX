# FolderCleanerX

A small Windows background tool that periodically empties a folder of your choice.

## What it does

1. On first run, it asks whether it should start automatically with Windows.
2. It opens the native Windows folder picker so you can choose which folder to clean.
3. It shows two warning dialogs you must both confirm with "Yes", acknowledging that the folder's contents will be permanently deleted, every 2 hours.
4. It saves your choice to a config file.
5. From then on, it runs quietly in the background (no console window) and shows a small folder icon in the system tray (bottom-right notification area).
6. It empties the selected folder — all files and subfolders inside it are deleted, but the folder itself is kept — every **2 hours** (and once immediately on startup).

## Tray menu

Right-click (or left-click, depending on Windows version) the tray icon to:

- **Folder: ...** — shows the currently configured folder (display only)
- **Change folder...** — opens the folder picker to select a different folder to clean (requires the same two-step confirmation)
- **Pause / Resume** — temporarily stop or resume the automatic cleanup
- **Clean now** — run the cleanup immediately
- **Quit** — exit the program

## Configuration

Settings are stored in:

```
%AppData%\Roaming\FolderCleanerX\config.json
```

To change the target folder or autostart behavior, delete this file and run the program again — the setup dialogs will appear once more.

## Autostart

If enabled, the program registers itself in:

```
HKEY_CURRENT_USER\Software\Microsoft\Windows\CurrentVersion\Run
```

under the name `FolderCleanerX`. Disable autostart by answering "No" during setup (after deleting the config file), or by removing the registry entry manually.

## Building

Requires Go 1.21+.

```sh
go build -ldflags="-H=windowsgui" -o folderCleanupX.exe .
```

The `-H=windowsgui` flag prevents a console window from appearing.

## GitHub Actions

A workflow at [`.github/workflows/build.yml`](.github/workflows/build.yml) builds the binary on `windows-latest` for every push to `main`, version tags (`v*`), pull requests, and manual triggers, and uploads `folderCleanupX.exe` as a build artifact.

## Warning

This tool **permanently deletes** the contents of the folder you select, with no confirmation and no undo. Choose the folder carefully (e.g. a downloads or temp folder), not anything containing important files.
