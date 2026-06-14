package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/getlantern/systray"
	"github.com/sqweek/dialog"
	"golang.org/x/sys/windows/registry"
)

const (
	appName         = "FolderCleanerX"
	version         = "1.1"
	cleanupInterval = 2 * time.Hour
)

var dialogTitle = fmt.Sprintf("%s v%s", appName, version)

type Config struct {
	FolderPath string `json:"folderPath"`
	Autostart  bool   `json:"autostart"`
	Configured bool   `json:"configured"`
}

var (
	cfgMu  sync.Mutex
	cfg    Config
	paused atomic.Bool

	nextMu      sync.Mutex
	nextCleanup time.Time
)

func setNextCleanup(t time.Time) {
	nextMu.Lock()
	nextCleanup = t
	nextMu.Unlock()
}

func nextCleanupLabel() string {
	if paused.Load() {
		return "Next cleanup: paused"
	}
	nextMu.Lock()
	t := nextCleanup
	nextMu.Unlock()
	if t.IsZero() {
		return "Next cleanup: -"
	}
	return "Next cleanup: " + t.Format("Mon 15:04")
}

func configFilePath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	appDir := filepath.Join(dir, appName)
	if err := os.MkdirAll(appDir, 0755); err != nil {
		return "", err
	}
	return filepath.Join(appDir, "config.json"), nil
}

func loadConfig() (Config, error) {
	var c Config
	path, err := configFilePath()
	if err != nil {
		return c, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return c, nil
		}
		return c, err
	}
	if err := json.Unmarshal(data, &c); err != nil {
		return c, err
	}
	return c, nil
}

func saveConfig(c Config) error {
	path, err := configFilePath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func setAutostart(enable bool) error {
	exePath, err := os.Executable()
	if err != nil {
		return err
	}

	key, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()

	if enable {
		return key.SetStringValue(appName, fmt.Sprintf(`"%s"`, exePath))
	}
	return key.DeleteValue(appName)
}

// cleanFolder removes all files and subfolders inside the given folder,
// but keeps the folder itself.
func cleanFolder(path string) error {
	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		full := filepath.Join(path, entry.Name())
		if err := os.RemoveAll(full); err != nil {
			return fmt.Errorf("failed to delete %s: %w", full, err)
		}
	}
	return nil
}

func firstTimeSetup() (Config, error) {
	var c Config

	autostart := dialog.Message("Should %s automatically run at Windows startup?", appName).
		Title(dialogTitle).
		YesNo()

	folder, err := dialog.Directory().Title("Select the folder to be cleaned every 2 hours").Browse()
	if err != nil {
		return c, fmt.Errorf("no folder selected: %w", err)
	}

	if !confirmFolderDeletion(folder) {
		return c, fmt.Errorf("folder deletion not confirmed")
	}

	c.FolderPath = folder
	c.Autostart = autostart
	c.Configured = true

	if err := setAutostart(autostart); err != nil {
		dialog.Message("Could not set up autostart: %v", err).Title(dialogTitle).Error()
	}

	if err := saveConfig(c); err != nil {
		return c, fmt.Errorf("could not save configuration: %w", err)
	}

	return c, nil
}

func runCleanup() {
	if paused.Load() {
		return
	}
	cfgMu.Lock()
	folder := cfg.FolderPath
	cfgMu.Unlock()

	if err := cleanFolder(folder); err != nil {
		dialog.Message("Error cleaning folder: %v", err).Title(dialogTitle).Error()
	}
}

func cleanupLoop() {
	runCleanup()
	setNextCleanup(time.Now().Add(cleanupInterval))

	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		runCleanup()
		setNextCleanup(time.Now().Add(cleanupInterval))
	}
}

func main() {
	ensureSingleInstance()

	var err error
	cfg, err = loadConfig()
	if err != nil {
		dialog.Message("Error loading configuration: %v", err).Title(dialogTitle).Error()
		os.Exit(1)
	}

	if !cfg.Configured || cfg.FolderPath == "" {
		cfg, err = firstTimeSetup()
		if err != nil {
			dialog.Message("Setup failed: %v", err).Title(dialogTitle).Error()
			os.Exit(1)
		}
		dialog.Message("Setup complete. The folder\n%s\nwill be emptied every 2 hours.", cfg.FolderPath).Title(dialogTitle).Info()
	}

	systray.Run(onReady, onExit)
}

func onReady() {
	systray.SetIcon(generateIcon())
	systray.SetTitle("")
	systray.SetTooltip(dialogTitle)

	cfgMu.Lock()
	folder := cfg.FolderPath
	cfgMu.Unlock()

	mFolder := systray.AddMenuItem(folderLabel(folder), "Currently cleaned folder")
	mFolder.Disable()
	mChange := systray.AddMenuItem("Change folder...", "Choose a different folder to clean")
	systray.AddSeparator()
	mNext := systray.AddMenuItem(nextCleanupLabel(), "Time of the next automatic cleanup")
	mNext.Disable()
	mPause := systray.AddMenuItem("Pause", "Pause automatic cleanup")
	mCleanNow := systray.AddMenuItem("Clean now", "Run cleanup immediately")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit", "Exit "+appName)

	go cleanupLoop()

	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			mNext.SetTitle(nextCleanupLabel())
		}
	}()

	go func() {
		for {
			select {
			case <-mChange.ClickedCh:
				folder, err := dialog.Directory().Title("Select the folder to be cleaned every 2 hours").Browse()
				if err != nil {
					continue // user cancelled
				}

				if !confirmFolderDeletion(folder) {
					continue
				}

				cfgMu.Lock()
				cfg.FolderPath = folder
				err = saveConfig(cfg)
				cfgMu.Unlock()

				if err != nil {
					dialog.Message("Could not save configuration: %v", err).Title(dialogTitle).Error()
					continue
				}
				mFolder.SetTitle(folderLabel(folder))

			case <-mPause.ClickedCh:
				if paused.Load() {
					paused.Store(false)
					mPause.SetTitle("Pause")
				} else {
					paused.Store(true)
					mPause.SetTitle("Resume")
				}
				mNext.SetTitle(nextCleanupLabel())

			case <-mCleanNow.ClickedCh:
				go func() {
					runCleanup()
					mNext.SetTitle(nextCleanupLabel())
				}()

			case <-mQuit.ClickedCh:
				systray.Quit()
				return
			}
		}
	}()
}

func onExit() {}

func folderLabel(folder string) string {
	return "Folder: " + folder
}
