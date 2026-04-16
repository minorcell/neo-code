//go:build !windows

package config

// PersistUserEnvVar persists a key/value pair into user-level environment storage.
// On non-Windows platforms, NeoCode currently relies on .env persistence and process env.
func PersistUserEnvVar(key string, value string) error {
	return nil
}
