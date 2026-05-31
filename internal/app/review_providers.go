package app

import (
	"context"

	"ero/internal/ports"
)

func buildReviewProviders(ctx context.Context, loader ports.ReviewProviderLoader) ([]ports.ReviewProviderClient, error) {
	if loader == nil {
		return nil, nil
	}
	return loader.LoadReviewProviders(ctx)
}
