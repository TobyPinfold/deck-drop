package backend

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// Serve runs a backend with config until ctx is cancelled, then stops it.
func Serve(ctx context.Context, logger *slog.Logger, config Config) error {
	service := New(logger, config)
	if err := service.Start(); err != nil {
		return fmt.Errorf("start backend: %w", err)
	}

	if config.PairOnStart && config.Pairing != nil {
		code, err := config.Pairing.NewCode()
		if err != nil {
			return fmt.Errorf("start pairing: %w", err)
		}
		logger.Info("pairing code (enter it on the other device)", "code", code.Value, "expires", code.ExpiresAt.Format(time.TimeOnly))
	}

	<-ctx.Done()

	if err := service.Stop(); err != nil {
		return fmt.Errorf("stop backend: %w", err)
	}
	return nil
}
