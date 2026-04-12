package state

import (
	"context"

	"neo-code/internal/config"
	providertypes "neo-code/internal/provider/types"
)

// Selection 表示当前激活的 provider 和 model。
type Selection struct {
	ProviderID string `json:"provider_id"`
	ModelID    string `json:"model_id"`
}

// ProviderOption 表示可供 UI 选择的 provider 及其模型候选。
type ProviderOption struct {
	ID     string                          `json:"id"`
	Name   string                          `json:"name"`
	Models []providertypes.ModelDescriptor `json:"models,omitempty"`
}

// Service 是对 config.SelectionService 的兼容封装，提供 state 命名空间接口。
type Service struct {
	legacy *config.SelectionService
}

// NewService 创建选择状态服务，并复用现有 config.SelectionService 能力。
func NewService(manager *config.Manager, supporters config.DriverSupporter, catalogs config.ModelCatalog) *Service {
	return &Service{
		legacy: config.NewSelectionService(manager, supporters, catalogs),
	}
}

// ListProviderOptions 返回当前运行时可见的 provider 列表及模型候选。
func (s *Service) ListProviderOptions(ctx context.Context) ([]ProviderOption, error) {
	items, err := s.legacy.ListProviders(ctx)
	if err != nil {
		return nil, err
	}
	options := make([]ProviderOption, 0, len(items))
	for _, item := range items {
		options = append(options, ProviderOption{
			ID:     item.ID,
			Name:   item.Name,
			Models: providertypes.CloneModelDescriptors(item.Models),
		})
	}
	return options, nil
}

// SelectProvider 切换当前 provider，并返回切换后的选择状态。
func (s *Service) SelectProvider(ctx context.Context, providerID string) (Selection, error) {
	selection, err := s.legacy.SelectProvider(ctx, providerID)
	if err != nil {
		return Selection{}, err
	}
	return Selection{
		ProviderID: selection.ProviderID,
		ModelID:    selection.ModelID,
	}, nil
}

// ListModels 返回当前 provider 的模型列表。
func (s *Service) ListModels(ctx context.Context) ([]providertypes.ModelDescriptor, error) {
	return s.legacy.ListModels(ctx)
}

// ListModelsSnapshot 返回当前 provider 的快照模型列表。
func (s *Service) ListModelsSnapshot(ctx context.Context) ([]providertypes.ModelDescriptor, error) {
	return s.legacy.ListModelsSnapshot(ctx)
}

// SetCurrentModel 切换当前模型，并返回切换后的选择状态。
func (s *Service) SetCurrentModel(ctx context.Context, modelID string) (Selection, error) {
	selection, err := s.legacy.SetCurrentModel(ctx, modelID)
	if err != nil {
		return Selection{}, err
	}
	return Selection{
		ProviderID: selection.ProviderID,
		ModelID:    selection.ModelID,
	}, nil
}

// EnsureSelection 确保当前 provider/model 选择有效，必要时执行修正。
func (s *Service) EnsureSelection(ctx context.Context) (Selection, error) {
	selection, err := s.legacy.EnsureSelection(ctx)
	if err != nil {
		return Selection{}, err
	}
	return Selection{
		ProviderID: selection.ProviderID,
		ModelID:    selection.ModelID,
	}, nil
}
