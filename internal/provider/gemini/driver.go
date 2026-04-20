package gemini

import (
	"context"
	"net/http"
	"time"

	"neo-code/internal/provider"
	"neo-code/internal/provider/openaicompat"
	providertypes "neo-code/internal/provider/types"
)

// DriverName 是 Gemini 协议驱动的唯一标识。
const DriverName = provider.DriverGemini

// Driver 返回 Gemini 协议驱动定义。
func Driver() provider.DriverDefinition {
	return provider.DriverDefinition{
		Name: DriverName,
		Build: func(ctx context.Context, cfg provider.RuntimeConfig) (provider.Provider, error) {
			return New(cfg)
		},
		Discover: func(ctx context.Context, cfg provider.RuntimeConfig) ([]providertypes.ModelDescriptor, error) {
			httpClient := &http.Client{
				Timeout: 90 * time.Second,
			}
			requestCfg, err := openaicompat.RequestConfigFromRuntime(cfg)
			if err != nil {
				return nil, err
			}
			return openaicompat.DiscoverModelDescriptors(ctx, httpClient, requestCfg)
		},
		ValidateCatalogIdentity: validateCatalogIdentity,
	}
}

// validateCatalogIdentity 在 SDK 模式下不再限制 endpoint 相关字段。
func validateCatalogIdentity(identity provider.ProviderIdentity) error {
	_ = identity
	return nil
}
