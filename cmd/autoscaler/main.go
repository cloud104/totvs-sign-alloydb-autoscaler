package main

import (
	"context"
	"fmt"
	"runtime"
	"time"

	monitoring "cloud.google.com/go/monitoring/apiv3/v2"
	"dev.azure.com/totvstfs/TOTVSApps-Infrastructure/_git/alloydb-autoscaler/internal/config"
	"dev.azure.com/totvstfs/TOTVSApps-Infrastructure/_git/alloydb-autoscaler/internal/log"
	"dev.azure.com/totvstfs/TOTVSApps-Infrastructure/_git/alloydb-autoscaler/internal/metrics"
	"dev.azure.com/totvstfs/TOTVSApps-Infrastructure/_git/alloydb-autoscaler/internal/scaling"
)

const AppName = "AlloyDB Autoscaler"

func main() {
	log.Info().
		Str("component", "app").
		Str("action", "startup").
		Str("appName", AppName).
		Str("version", runtime.Version()).
		Msg("Application started successfully")

	ctx := context.Background()
	client, err := monitoring.NewMetricClient(ctx)
	if err != nil {
		log.Fatal().
			Str("component", "app").
			Str("action", "initialize").
			Err(err).
			Msg("Failed to create metrics client")
	}
	defer client.Close()

	var scaleUpCount, scaleDownCount int
	evaluationStart := time.Now()
	cycleCount := 0

	for {
		cycleCount++
		cycleStartTime := time.Now()
		
		func() {
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.Get().TimeoutSeconds)*time.Second)
			defer cancel()

			log.Debug().
				Str("component", "app").
				Str("action", "check").
				Int("cycle", cycleCount).
				Msg("Starting metrics check cycle")

			if err := metrics.CheckMetrics(ctx, client, &scaleUpCount, &scaleDownCount); err != nil {
				if ctx.Err() == context.DeadlineExceeded {
					log.ErrorMessage("Metrics check timeout").
						Str("component", "app").
						Str("action", "check").
						Int("timeoutSeconds", config.Get().TimeoutSeconds).
						Int("cycle", cycleCount).
						Send()
				} else {
					log.Error(err).
						Str("component", "app").
						Str("action", "check").
						Int("cycle", cycleCount).
						Msg("Error checking metrics")
				}
			}
			
			log.Debug().
				Str("component", "app").
				Str("action", "check").
				Int("cycle", cycleCount).
				Str("duration", fmt.Sprintf("%.2fs", time.Since(cycleStartTime).Seconds())).
				Msg("Metrics check cycle completed")
		}()

		evalElapsed := time.Since(evaluationStart)
		if evalElapsed >= time.Duration(config.Get().Evaluation)*time.Second {
			log.Info().
				Str("component", "scaling").
				Str("action", "decision").
				Int("scaleUpVotes", scaleUpCount).
				Int("scaleDownVotes", scaleDownCount).
				Str("evaluationPeriod", fmt.Sprintf("%.2fs", evalElapsed.Seconds())).
				Msg("Making scaling decision")
				
			if scaleUpCount > scaleDownCount && scaleUpCount > 0 {
				if err := scaling.ScaleUp(context.Background()); err != nil {
					log.Error(err).
						Str("component", "scaling").
						Str("action", "scaleUp").
						Msg("Failed to scale up replicas")
				} else {
					log.Info().
						Str("component", "scaling").
						Str("action", "scaleUp").
						Msg("Scale up operation completed successfully")
				}
			} else if scaleDownCount > scaleUpCount && scaleDownCount > 0 {
				if err := scaling.ScaleDown(context.Background()); err != nil {
					log.Error(err).
						Str("component", "scaling").
						Str("action", "scaleDown").
						Msg("Failed to scale down replicas")
				} else {
					log.Info().
						Str("component", "scaling").
						Str("action", "scaleDown").
						Msg("Scale down operation completed successfully")
				}
			} else {
				log.Info().
					Str("component", "scaling").
					Str("action", "maintain").
					Msg("No scaling action needed, maintaining current replica count")
			}

			scaleUpCount = 0
			scaleDownCount = 0
			evaluationStart = time.Now()
		}

		logTimer(config.Get().CheckInterval, cycleCount)
	}
}

func logTimer(duration int, cycle int) {
	nextCheck := time.Now().Add(time.Duration(duration) * time.Second)
	log.Debug().
		Str("component", "app").
		Str("action", "schedule").
		Int("cycle", cycle).
		Str("nextCheckTime", nextCheck.Format("15:04:05")).
		Int("intervalSeconds", duration).
		Msg("Next metrics check scheduled")
	time.Sleep(time.Duration(duration) * time.Second)
}
