package config

import (
	"os"
	"path/filepath"
)

func appDir() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Dir(exe), nil
}

func ConfigDir() (string, error) {
	return appDir()
}

func ConfigFile() (string, error) {
	dir, err := appDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.yaml"), nil
}

func DataDir() (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return appDir()
	}
	return filepath.Join(base, "monorhyme-search"), nil
}

func DefaultDBPath() (string, error) {
	dir, err := DataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "index.db"), nil
}
