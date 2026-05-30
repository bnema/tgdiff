package main

import (
	"bytes"
	"context"
	"testing"

	"ero/pkg/plugin"
)

func TestPimonoCapabilities(t *testing.T) {
	provider := pimonoProvider{}
	result, err := provider.Initialize(context.Background(), plugin.InitializeRequest{Protocol: plugin.ProtocolVersion})
	if err != nil {
		t.Fatalf("Initialize returned error: %v", err)
	}
	if !result.Provider.Capabilities.PublishReview || result.Provider.Capabilities.LoadRemoteComments {
		t.Fatalf("unexpected capabilities: %#v", result.Provider.Capabilities)
	}
}

func TestPublishReviewDryRunWritesPayload(t *testing.T) {
	var stderr bytes.Buffer
	provider := pimonoProvider{stderr: &stderr, getenv: func(key string) string {
		if key == "PIMONO_DRY_RUN" {
			return "1"
		}
		return ""
	}}
	result, err := provider.PublishReview(context.Background(), plugin.PublishReviewParams{Payload: plugin.ReviewPublishPayload{Context: plugin.ReviewContext{Session: plugin.ReviewSessionMetadata{LocalReviewID: "local"}}}})
	if err != nil {
		t.Fatalf("PublishReview returned error: %v", err)
	}
	if result.Result.ProviderID != providerID || stderr.Len() == 0 {
		t.Fatalf("expected dry-run result and stderr payload, got result=%#v stderr=%q", result, stderr.String())
	}
}
