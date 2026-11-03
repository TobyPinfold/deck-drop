package backend

import (
	"context"
	"fmt"
	"log/slog"
)

// Serve runs a backend with config until ctx is cancelled, then stops it.
func Serve(ctx context.Context, logger *slog.Logger, config Config) error {
	service := New(logger, config)
	if err := service.Start(); err != nil {
		return fmt.Errorf("start backend: %w", err)
	}

	<-ctx.Done()

	if err := service.Stop(); err != nil {
		return fmt.Errorf("stop backend: %w", err)
	}
	return nil
}
