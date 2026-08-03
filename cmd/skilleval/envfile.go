package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

// loadDotEnv loads KEY=VALUE pairs from path into the process environment.
// A missing file is a no-op. Already-set process environment variables win.
func loadDotEnv(path string) error {
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("envfile: %w", err)
	}
	if err := godotenv.Load(path); err != nil {
		return fmt.Errorf("envfile: %s: %w", path, err)
	}
	return nil
}
