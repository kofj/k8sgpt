package ai

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/k8sgpt-ai/k8sgpt/pkg/cache"
	"github.com/k8sgpt-ai/k8sgpt/pkg/util"
	"github.com/sashabaranov/go-openai"
)

const netdAPIURLv1 = "https://netd.fun/v1"

type NetdAIClient struct {
	OpenAIClient
}

func (a *NetdAIClient) GetName() string {
	return "netd"
}

func (n *NetdAIClient) Configure(config IAIConfig, lang string) error {
	token := config.GetPassword()
	// ignore config engine
	// ignore config baseURL

	var netdConfig = openai.DefaultConfig(token)
	netdConfig.BaseURL = netdAPIURLv1

	client := openai.NewClientWithConfig(netdConfig)
	if client == nil {
		return errors.New("error creating NETD.FUN OpenAI client")
	}
	n.language = lang
	n.client = client
	n.model = config.GetModel()
	return nil
}

func (n *NetdAIClient) GetCompletion(ctx context.Context, prompt string) (string, error) {
	// Create a completion request
	resp, err := n.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: n.model,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    "user",
				Content: fmt.Sprintf(default_prompt, n.language, prompt),
			},
		},
	})
	if err != nil {
		return "", err
	}
	return resp.Choices[0].Message.Content, nil
}

func (n *NetdAIClient) Parse(ctx context.Context, prompt []string, cache cache.ICache) (string, error) {
	// parse the text with the AI backend
	inputKey := strings.Join(prompt, " ")
	// Check for cached data
	sEnc := base64.StdEncoding.EncodeToString([]byte(inputKey))
	cacheKey := util.GetCacheKey(n.GetName(), n.language, sEnc)

	if !cache.IsCacheDisabled() && cache.Exists(cacheKey) {
		response, err := cache.Load(cacheKey)
		if err != nil {
			return "", err
		}

		if response != "" {
			output, err := base64.StdEncoding.DecodeString(response)
			if err != nil {
				color.Red("error decoding cached data: %v", err)
				return "", nil
			}
			return string(output), nil
		}
	}

	response, err := n.GetCompletion(ctx, inputKey)
	if err != nil {
		color.Red("error getting completion: %v", err)
		return "", err
	}

	err = cache.Store(cacheKey, base64.StdEncoding.EncodeToString([]byte(response)))

	if err != nil {
		color.Red("error storing value to cache: %v", err)
		return "", nil
	}

	return response, nil
}
