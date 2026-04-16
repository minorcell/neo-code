//go:build windows

package config

import (
	"errors"
	"fmt"
	"strings"

	"golang.org/x/sys/windows/registry"
)

const windowsUserEnvironmentKey = `Environment`

// PersistUserEnvVar persists key/value into Windows user environment variables.
func PersistUserEnvVar(key string, value string) error {
	normalizedKey := strings.TrimSpace(key)
	if normalizedKey == "" {
		return errors.New("config: env key is empty")
	}
	if strings.ContainsAny(normalizedKey, " \t\r\n=") {
		return fmt.Errorf("config: env key %q is invalid", normalizedKey)
	}
	if strings.ContainsAny(value, "\r\n") {
		return errors.New("config: env value contains newline")
	}

	envKey, _, err := registry.CreateKey(registry.CURRENT_USER, windowsUserEnvironmentKey, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("config: open windows user env: %w", err)
	}
	defer envKey.Close()

	if err := envKey.SetStringValue(normalizedKey, value); err != nil {
		return fmt.Errorf("config: set windows user env %q: %w", normalizedKey, err)
	}
	return nil
}
