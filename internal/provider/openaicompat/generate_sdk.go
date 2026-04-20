package openaicompat

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	openai "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"

	"neo-code/internal/provider"
	"neo-code/internal/provider/openaicompat/chatcompletions"
	"neo-code/internal/provider/openaicompat/responses"
	providertypes "neo-code/internal/provider/types"
)

// generateSDKChatCompletions 走 SDK chat/completions 发送请求，复用本地 wire 解析。
func (p *Provider) generateSDKChatCompletions(
	ctx context.Context,
	req providertypes.GenerateRequest,
	events chan<- providertypes.StreamEvent,
) error {
	payload, err := chatcompletions.BuildRequest(ctx, p.cfg, req)
	if err != nil {
		return err
	}
	endpoint, err := resolveChatEndpoint(p.cfg)
	if err != nil {
		return err
	}
	return p.sendSDKStreamRequest(ctx, endpoint, payload, chatcompletions.ConsumeStream, ParseError, events)
}

// generateSDKResponses 走 SDK responses 发送请求，复用本地流事件映射。
func (p *Provider) generateSDKResponses(
	ctx context.Context,
	req providertypes.GenerateRequest,
	events chan<- providertypes.StreamEvent,
) error {
	payload, err := responses.BuildRequest(ctx, p.cfg, req)
	if err != nil {
		return err
	}
	endpoint, err := resolveChatEndpoint(p.cfg)
	if err != nil {
		return err
	}
	return p.sendSDKStreamRequest(ctx, endpoint, payload, responses.ConsumeStream, ParseError, events)
}

func (p *Provider) sendSDKStreamRequest(
	ctx context.Context,
	endpoint string,
	payload any,
	consumeStream func(context.Context, io.Reader, chan<- providertypes.StreamEvent) error,
	parseError func(*http.Response) error,
	events chan<- providertypes.StreamEvent,
) error {
	client := p.newSDKClient()
	var resp *http.Response

	err := client.Post(
		ctx,
		strings.TrimSpace(endpoint),
		payload,
		nil,
		option.WithResponseInto(&resp),
		option.WithHeader("Accept", "text/event-stream"),
	)
	if err != nil {
		if resp != nil && resp.StatusCode >= http.StatusBadRequest {
			return parseError(resp)
		}
		return fmt.Errorf("%ssend request: %w", errorPrefix, err)
	}
	if resp == nil {
		return fmt.Errorf("%ssend request: empty response", errorPrefix)
	}
	defer func(body io.ReadCloser) {
		if closeErr := body.Close(); closeErr != nil {
			log.Printf("%sclose response body: %v", errorPrefix, closeErr)
		}
	}(resp.Body)

	if resp.StatusCode >= http.StatusBadRequest {
		return parseError(resp)
	}
	return consumeStream(ctx, resp.Body, events)
}

func (p *Provider) newSDKClient() openai.Client {
	return openai.NewClient(
		option.WithHTTPClient(p.client),
		option.WithAPIKey(strings.TrimSpace(p.cfg.APIKey)),
	)
}

func resolveChatEndpoint(cfg provider.RuntimeConfig) (string, error) {
	endpoint, err := provider.ResolveChatEndpointURL(cfg.BaseURL, cfg.ChatEndpointPath)
	if err != nil {
		return "", fmt.Errorf("%sinvalid chat endpoint configuration: %w", errorPrefix, err)
	}
	return endpoint, nil
}
