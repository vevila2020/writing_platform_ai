package modelio

import (
	"context"
	"fmt"
	"net/http"

	"github.com/vevila2020/writing_platform_ai/config"
	"github.com/vevila2020/writing_platform_ai/internal/hostruntime"

	agentmodel "github.com/vevila2020/writing_platform_ai/agent/model"
	"github.com/vevila2020/writing_platform_ai/agent/model/providers"
)

// ConfigForAgent resolves the provider-neutral model configuration for one
// Agent kind.
func ConfigForAgent(cfg *config.Config, agentKind string) (providers.ModelConfig, error) {
	return ConfigFromResolved(config.ResolveAgentModel(cfg, agentKind))
}

func ConfigFromResolved(resolved config.ResolvedModelSettings) (providers.ModelConfig, error) {
	provider := providers.ProviderID(resolved.Provider)
	protocol := providers.ProtocolID(resolved.Protocol)
	if provider == "" {
		provider = providers.ProviderOpenAICompatible
	}
	modelConfig := providers.ModelConfig{
		Provider:          provider,
		Protocol:          protocol,
		APIKey:            resolved.APIKey,
		Model:             resolved.Model,
		BaseURL:           resolved.BaseURL,
		Headers:           resolved.Headers,
		SessionKeyMapping: resolved.SessionKeyMapping,
		MaxOutputTokens:   resolved.MaxTokens,
		ThinkingLevel:     providers.ThinkingLevel(resolved.ThinkingLevel),
		HTTPClient:        modelHTTPClient(),
	}
	if len(resolved.ProtocolOptions) != 0 {
		options, err := providers.EncodeProtocolOptions(resolved.ProtocolOptions)
		if err != nil {
			return providers.ModelConfig{}, fmt.Errorf("convert model protocol options: %w", err)
		}
		modelConfig.ProtocolOptions = options
	}
	if resolved.Temperature != nil {
		temperature := float32(*resolved.Temperature)
		modelConfig.Temperature = &temperature
	}
	return modelConfig, nil
}

// modelHTTPClient uses the host's proxy policy for every model request. The
// selector honors explicit environment overrides and otherwise reads the OS
// system proxy, so native and installed builds behave consistently. Returning
// nil from the selector keeps local and directly configured endpoints working.
func modelHTTPClient() *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = hostruntime.NewHTTPProxy()
	return &http.Client{Transport: transport}
}

func NewChatModel(ctx context.Context, config providers.ModelConfig) (agentmodel.ToolCallingChatModel, error) {
	return defaultRuntime.NewChatModel(ctx, config)
}

func WithJSONObjectOutput(config providers.ModelConfig) providers.ModelConfig {
	config.OutputFormat = &providers.OutputFormat{Type: providers.OutputFormatJSONObject}
	return config
}
