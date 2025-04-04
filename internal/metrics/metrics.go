package metrics

import (
	"context"
	"fmt"
	"math"
	"time"

	monitoring "cloud.google.com/go/monitoring/apiv3/v2"
	"cloud.google.com/go/monitoring/apiv3/v2/monitoringpb"
	"github.com/heraque/alloydb-autoscaler/internal/alloydb"
	"github.com/heraque/alloydb-autoscaler/internal/config"
	"github.com/heraque/alloydb-autoscaler/internal/log"
	"google.golang.org/api/iterator"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// CheckMetrics checks AlloyDB metrics and updates scaling counters
func CheckMetrics(ctx context.Context, client *monitoring.MetricClient, currentScaleUpCount, currentScaleDownCount int) (int, int, error) {
	startTime := time.Now()

	memoryFreeBytes, err := QueryMetric(ctx, client, config.Get().MemoryMetric)
	if err != nil {
		return 0, 0, fmt.Errorf("error querying free memory: %w", err)
	}

	cpuUsage, err := QueryMetric(ctx, client, config.Get().CPUMetric)
	if err != nil {
		return 0, 0, fmt.Errorf("error querying CPU usage: %w", err)
	}

	totalMemoryGB, err := alloydb.GetTotalMemory(ctx)
	if err != nil {
		return 0, 0, fmt.Errorf("error getting total memory: %w", err)
	}

	memoryFreeGB := memoryFreeBytes / (1024 * 1024 * 1024)
	memoryUsedGB := totalMemoryGB - memoryFreeGB
	memoryUsagePercent := (memoryUsedGB / totalMemoryGB) * 100

	cpuUsagePercent := cpuUsage * 100

	log.Debug().
		Str("component", "metrics").
		Str("action", "collect").
		Str("instance", config.Get().InstanceName).
		Str("cluster", config.Get().ClusterName).
		Float64("cpuUsage", math.Round(cpuUsagePercent*100)/100).
		Str("memoryUsage", fmt.Sprintf("%.2f%%", math.Round(memoryUsagePercent*100)/100)).
		Str("cpuThreshold", fmt.Sprintf("%.2f%%", config.Get().CPUThreshold)).
		Str("memoryThreshold", fmt.Sprintf("%.2f%%", config.Get().MemoryThreshold)).
		Str("duration", fmt.Sprintf("%.2fs", time.Since(startTime).Seconds())).
		Msg("AlloyDB resource metrics collected")

	currentCount, err := alloydb.GetReadPoolNodeCount(ctx)
	if err != nil {
		return 0, 0, err
	}

	newScaleUpCount := currentScaleUpCount
	newScaleDownCount := currentScaleDownCount

	if memoryUsagePercent > config.Get().MemoryThreshold || cpuUsagePercent > config.Get().CPUThreshold {
		if currentCount < config.Get().MaxReplicas {
			log.Info().
				Str("component", "scaling").
				Str("action", "evaluate").
				Str("instance", config.Get().InstanceName).
				Float64("cpuUsage", math.Round(cpuUsagePercent*100)/100).
				Str("memoryUsage", fmt.Sprintf("%.2f%%", math.Round(memoryUsagePercent*100)/100)).
				Str("cpuThreshold", fmt.Sprintf("%.2f%%", config.Get().CPUThreshold)).
				Str("memoryThreshold", fmt.Sprintf("%.2f%%", config.Get().MemoryThreshold)).
				Int("currentReplicas", currentCount).
				Int("maxReplicas", config.Get().MaxReplicas).
				Int("scaleUpVotes", newScaleUpCount+1).
				Msg("Insufficient resources detected, considering scaling up")
			newScaleUpCount++
			newScaleDownCount = 0
		} else {
			log.Warn().
				Str("component", "scaling").
				Str("action", "evaluate").
				Str("instance", config.Get().InstanceName).
				Int("currentReplicas", currentCount).
				Int("maxReplicas", config.Get().MaxReplicas).
				Msg("Insufficient resources detected, but maximum replicas limit reached")
		}
	} else if currentCount > config.Get().MinReplicas {
		log.Info().
			Str("component", "scaling").
			Str("action", "evaluate").
			Str("instance", config.Get().InstanceName).
			Float64("cpuUsage", math.Round(cpuUsagePercent*100)/100).
			Str("memoryUsage", fmt.Sprintf("%.2f%%", math.Round(memoryUsagePercent*100)/100)).
			Str("cpuThreshold", fmt.Sprintf("%.2f%%", config.Get().CPUThreshold)).
			Str("memoryThreshold", fmt.Sprintf("%.2f%%", config.Get().MemoryThreshold)).
			Int("currentReplicas", currentCount).
			Int("minReplicas", config.Get().MinReplicas).
			Int("scaleDownVotes", newScaleDownCount+1).
			Msg("Excess resources detected, considering scaling down")
		newScaleDownCount++
		newScaleUpCount = 0
	} else {
		LogNormalResources(currentCount)
		newScaleUpCount = 0
		newScaleDownCount = 0
	}

	return newScaleUpCount, newScaleDownCount, nil
}

// QueryMetric queries a specific metric from Cloud Monitoring
func QueryMetric(ctx context.Context, client *monitoring.MetricClient, metricType string) (float64, error) {
	now := time.Now()
	startTime := now.Add(-5 * time.Minute)

	req := &monitoringpb.ListTimeSeriesRequest{
		Name:   fmt.Sprintf("projects/%s", config.Get().GCPProject),
		Filter: fmt.Sprintf(`metric.type = "%s" AND resource.labels.instance_id = "%s"`, metricType, config.Get().InstanceName),
		Interval: &monitoringpb.TimeInterval{
			StartTime: timestamppb.New(startTime),
			EndTime:   timestamppb.New(now),
		},
		View: monitoringpb.ListTimeSeriesRequest_FULL,
	}

	it := client.ListTimeSeries(ctx, req)
	var lastValue float64
	for {
		resp, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			if ctx.Err() == context.DeadlineExceeded {
				return 0, fmt.Errorf("timeout querying metric %s: %w", metricType, err)
			}
			return 0, fmt.Errorf("error iterating time series: %w", err)
		}
		if len(resp.Points) > 0 {
			switch v := resp.Points[0].Value.Value.(type) {
			case *monitoringpb.TypedValue_DoubleValue:
				lastValue = v.DoubleValue
			case *monitoringpb.TypedValue_Int64Value:
				lastValue = float64(v.Int64Value)
			default:
				return 0, fmt.Errorf("unsupported value type: %T", v)
			}
		}
	}

	return lastValue, nil
}

// LogNormalResources logs when resources are within normal thresholds
func LogNormalResources(count int) {
	log.Info().
		Str("component", "scaling").
		Str("action", "evaluate").
		Str("instance", config.Get().InstanceName).
		Int("currentReplicas", count).
		Int("minReplicas", config.Get().MinReplicas).
		Int("maxReplicas", config.Get().MaxReplicas).
		Msg("AlloyDB resources within normal thresholds")
}
