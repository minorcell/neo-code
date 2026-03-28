package gemini

import (
	"neo-code/internal/config"
	"neo-code/internal/provider/openai"
)

const (
	Name             = "gemini"
	DriverName       = openai.DriverName
	DefaultBaseURL   = "https://generativelanguage.googleapis.com/v1beta/openai"
	DefaultModel     = "gemini-2.5-flash"
	DefaultAPIKeyEnv = "GEMINI_API_KEY"
)

var builtinModels = []string{
	DefaultModel,
	"gemini-2.5-pro",
	"gemini-2.0-flash",
}

func BuiltinConfig() config.ProviderConfig {
	return config.ProviderConfig{
		Name:      Name,
		Driver:    DriverName,
		BaseURL:   DefaultBaseURL,
		Model:     DefaultModel,
		Models:    append([]string(nil), builtinModels...),
		APIKeyEnv: DefaultAPIKeyEnv,
	}
}
