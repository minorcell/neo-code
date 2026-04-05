package runtime

import (
	"context"
	"errors"
	"strings"

	"neo-code/internal/config"
	agentcontext "neo-code/internal/context"
	contextcompact "neo-code/internal/context/compact"
	"neo-code/internal/provider"
)

type compactSummaryGenerator struct {
	providerFactory ProviderFactory
	providerConfig  config.ResolvedProviderConfig
	model           string
}

func newCompactSummaryGenerator(
	providerFactory ProviderFactory,
	providerCfg config.ResolvedProviderConfig,
	model string,
) contextcompact.SummaryGenerator {
	return &compactSummaryGenerator{
		providerFactory: providerFactory,
		providerConfig:  providerCfg,
		model:           strings.TrimSpace(model),
	}
}

func (g *compactSummaryGenerator) Generate(ctx context.Context, input contextcompact.SummaryInput) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if g.providerFactory == nil {
		return "", errors.New("runtime: compact summary generator provider factory is nil")
	}
	if strings.TrimSpace(g.providerConfig.Driver) == "" ||
		strings.TrimSpace(g.providerConfig.BaseURL) == "" ||
		strings.TrimSpace(g.providerConfig.APIKey) == "" {
		return "", errors.New("runtime: compact summary generator provider config is incomplete")
	}

	prompt := agentcontext.BuildCompactPrompt(agentcontext.CompactPromptInput{
		Mode:                     string(input.Mode),
		ManualStrategy:           input.Config.ManualStrategy,
		ManualKeepRecentMessages: input.Config.ManualKeepRecentMessages,
		ArchivedMessageCount:     input.ArchivedMessageCount,
		MaxSummaryChars:          input.Config.MaxSummaryChars,
		ArchivedMessages:         input.ArchivedMessages,
		RetainedMessages:         input.RetainedMessages,
	})

	modelProvider, err := g.providerFactory.Build(ctx, g.providerConfig)
	if err != nil {
		return "", err
	}

	// 使用流式事件通道收集 compact 摘要响应。
	streamEvents := make(chan provider.StreamEvent, 32)
	streamDone := make(chan struct{})
	acc := newStreamAccumulator()

	go func() {
		defer close(streamDone)
		for {
			select {
			case event, ok := <-streamEvents:
				if !ok {
					return
				}
				switch event.Type {
				case provider.StreamEventTextDelta:
					if payload, ok := event.Payload.(provider.TextDeltaPayload); ok {
						acc.accumulateTextDelta(payload.Text)
					}
				case provider.StreamEventToolCallStart:
					if payload, ok := event.Payload.(provider.ToolCallStartPayload); ok {
						acc.accumulateToolCallStart(payload.Index, payload.ID, payload.Name)
					}
				case provider.StreamEventToolCallDelta:
					if payload, ok := event.Payload.(provider.ToolCallDeltaPayload); ok {
						acc.accumulateToolCallDelta(payload.Index, payload.ID, payload.ArgumentsDelta)
					}
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	err = modelProvider.Chat(ctx, provider.ChatRequest{
		Model:        g.model,
		SystemPrompt: prompt.SystemPrompt,
		Messages: []provider.Message{{
			Role:    provider.RoleUser,
			Content: prompt.UserPrompt,
		}},
	}, streamEvents)
	close(streamEvents)
	<-streamDone

	if err != nil {
		return "", err
	}

	message := acc.buildMessage()
	if len(message.ToolCalls) > 0 {
		return "", errors.New("runtime: compact summary response must not contain tool calls")
	}

	summary := strings.TrimSpace(message.Content)
	if summary == "" {
		return "", errors.New("runtime: compact summary response is empty")
	}
	return summary, nil
}
