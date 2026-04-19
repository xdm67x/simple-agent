package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const baseUrl = "https://openrouter.ai/api/v1"

type model struct {
	Id            string `json:"id"`
	CanonicalSlug string `json:"canonical_slug"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	ContextLength uint32 `json:"context_length"`
	InputCost     uint64 `json:"pricing.prompt"`
	OutputCost    uint64 `json:"pricing.completion"`
}

type modelsResponse struct {
	Data []model `json:"data"`
}

type apiKeyTransport struct {
	apiKey string

	http.RoundTripper
}

func (t *apiKeyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", t.apiKey))

	if t.RoundTripper == nil {
		t.RoundTripper = http.DefaultTransport
	}

	return t.RoundTripper.RoundTrip(req)
}

type OpenRouter struct {
	client *http.Client

	Models []model
}

func NewOpenRouterProvider(apiKey string) (OpenRouter, error) {
	provider := OpenRouter{
		client: &http.Client{
			Transport: &apiKeyTransport{
				apiKey: apiKey,
			},
		},
	}

	resp, err := provider.client.Get(fmt.Sprintf("%s/models?supported_parameters=tools", baseUrl))
	if err != nil {
		return provider, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return provider, err
	}

	var response modelsResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return provider, err
	}

	provider.Models = response.Data
	return provider, nil
}
