package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	dir := filepath.Join(home, ".config", "cash5")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}

	return filepath.Join(dir, "draws.json"), nil
}

func loadDraws() ([]Draw, error) {
	path, err := configPath()
	if err != nil {
		return nil, err
	}

	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []Draw{}, nil
		}
		return nil, err
	}
	defer file.Close()

	var draws []Draw
	err = json.NewDecoder(file).Decode(&draws)
	return draws, err
}

func saveDraws(draws []Draw) error {
	path, err := configPath()
	if err != nil {
		return err
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	return enc.Encode(draws)
}
