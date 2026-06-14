package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sqweek/dialog"
	"golang.org/x/sys/windows/registry"
)

const (
	appName         = "FolderCleanerX"
	cleanupInterval = 2 * time.Hour
)

type Config struct {
	FolderPath string `json:"folderPath"`
	Autostart  bool   `json:"autostart"`
	Configured bool   `json:"configured"`
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
	var cfg Config
	path, err := configFilePath()
	if err != nil {
		return cfg, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func saveConfig(cfg Config) error {
	path, err := configFilePath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
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
	var cfg Config

	autostart := dialog.Message("Should %s automatically run at Windows startup?", appName).
		Title(appName).
		YesNo()

	folder, err := dialog.Directory().Title("Select the folder to be cleaned every 2 hours").Browse()
	if err != nil {
		return cfg, fmt.Errorf("no folder selected: %w", err)
	}

	cfg.FolderPath = folder
	cfg.Autostart = autostart
	cfg.Configured = true

	if err := setAutostart(autostart); err != nil {
		dialog.Message("Could not set up autostart: %v", err).Title(appName).Error()
	}

	if err := saveConfig(cfg); err != nil {
		return cfg, fmt.Errorf("could not save configuration: %w", err)
	}

	return cfg, nil
}

func main() {
	cfg, err := loadConfig()
	if err != nil {
		dialog.Message("Error loading configuration: %v", err).Title(appName).Error()
		os.Exit(1)
	}

	if !cfg.Configured || cfg.FolderPath == "" {
		cfg, err = firstTimeSetup()
		if err != nil {
			dialog.Message("Setup failed: %v", err).Title(appName).Error()
			os.Exit(1)
		}
		dialog.Message("Setup complete. The folder\n%s\nwill be emptied every 2 hours.", cfg.FolderPath).Title(appName).Info()
	}

	// Clean immediately once, then on the regular interval.
	if err := cleanFolder(cfg.FolderPath); err != nil {
		dialog.Message("Error cleaning folder: %v", err).Title(appName).Error()
	}

	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		if err := cleanFolder(cfg.FolderPath); err != nil {
			dialog.Message("Error cleaning folder: %v", err).Title(appName).Error()
		}
	}
}
