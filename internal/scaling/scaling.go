package scaling

import (
	"context"
	"fmt"
	"time"

	"github.com/heraque/alloydb-autoscaler/tree/main/internal/alloydb"
	"github.com/heraque/alloydb-autoscaler/tree/main/internal/config"
	"github.com/heraque/alloydb-autoscaler/tree/main/internal/log"
)

// ScaleUp aumenta o número de réplicas em 1, se possível
func ScaleUp(ctx context.Context) error {
	startTime := time.Now()

	currentCount, err := alloydb.GetReadPoolNodeCount(ctx)
	if err != nil {
		return err
	}

	if currentCount < config.Get().MaxReplicas {
		newCount := currentCount + 1

		log.Info().
			Str("component", "scaling").
			Str("action", "scaleUp").
			Str("instance", config.Get().InstanceName).
			Int("currentReplicas", currentCount).
			Int("targetReplicas", newCount).
			Int("maxReplicas", config.Get().MaxReplicas).
			Msg("Initiating scale up operation")

		operation, err := alloydb.UpdateReplicaCount(ctx, newCount)
		if err != nil {
			return err
		}

		err = alloydb.WaitForOperation(ctx, operation)
		if err != nil {
			return fmt.Errorf("error waiting for scale up operation to complete: %w", err)
		}

		log.Info().
			Str("component", "scaling").
			Str("action", "scaleUp").
			Str("instance", config.Get().InstanceName).
			Int("newReplicaCount", newCount).
			Dur("duration", time.Since(startTime).Round(time.Second)).
			Msg("Scale up operation completed successfully")
	} else {
		log.Warn().
			Str("component", "scaling").
			Str("action", "scaleUp").
			Str("instance", config.Get().InstanceName).
			Int("currentReplicas", currentCount).
			Int("maxReplicas", config.Get().MaxReplicas).
			Msg("Maximum replica count reached, cannot scale up further")
	}
	return nil
}

// ScaleDown diminui o número de réplicas em 1, se possível
func ScaleDown(ctx context.Context) error {
	startTime := time.Now()

	currentCount, err := alloydb.GetReadPoolNodeCount(ctx)
	if err != nil {
		return err
	}

	if currentCount > config.Get().MinReplicas {
		newCount := currentCount - 1

		log.Info().
			Str("component", "scaling").
			Str("action", "scaleDown").
			Str("instance", config.Get().InstanceName).
			Int("currentReplicas", currentCount).
			Int("targetReplicas", newCount).
			Int("minReplicas", config.Get().MinReplicas).
			Msg("Initiating scale down operation")

		operation, err := alloydb.UpdateReplicaCount(ctx, newCount)
		if err != nil {
			return err
		}

		err = alloydb.WaitForOperation(ctx, operation)
		if err != nil {
			return fmt.Errorf("error waiting for scale down operation to complete: %w", err)
		}

		log.Info().
			Str("component", "scaling").
			Str("action", "scaleDown").
			Str("instance", config.Get().InstanceName).
			Int("newReplicaCount", newCount).
			Dur("duration", time.Since(startTime).Round(time.Second)).
			Msg("Scale down operation completed successfully")
	} else {
		log.Warn().
			Str("component", "scaling").
			Str("action", "scaleDown").
			Str("instance", config.Get().InstanceName).
			Int("currentReplicas", currentCount).
			Int("minReplicas", config.Get().MinReplicas).
			Msg("Minimum replica count reached, cannot scale down further")
	}
	return nil
}
