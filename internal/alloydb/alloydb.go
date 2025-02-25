package alloydb

import (
	"context"
	"fmt"
	"time"

	"dev.azure.com/totvstfs/TOTVSApps-Infrastructure/_git/alloydb-autoscaler/internal/config"
	"dev.azure.com/totvstfs/TOTVSApps-Infrastructure/_git/alloydb-autoscaler/internal/log"
	"google.golang.org/api/alloydb/v1"
	"google.golang.org/api/option"
)

// GetReadPoolNodeCount returns the current number of nodes in the read pool
func GetReadPoolNodeCount(ctx context.Context) (int, error) {
	service, err := alloydb.NewService(ctx, option.WithCredentialsFile(config.Get().GoogleApplicationCredentials))
	if err != nil {
		return 0, fmt.Errorf("error creating AlloyDB service: %w", err)
	}

	instanceName := fmt.Sprintf("projects/%s/locations/%s/clusters/%s/instances/%s", 
		config.Get().GCPProject, 
		config.Get().Region, 
		config.Get().ClusterName, 
		config.Get().InstanceName)
	instance, err := service.Projects.Locations.Clusters.Instances.Get(instanceName).Context(ctx).Do()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return 0, fmt.Errorf("timeout getting instance: %w", err)
		}
		return 0, fmt.Errorf("error getting instance: %w", err)
	}

	return int(instance.ReadPoolConfig.NodeCount), nil
}

// GetTotalMemory returns the total memory of the instance in GB
func GetTotalMemory(ctx context.Context) (float64, error) {
	service, err := alloydb.NewService(ctx, option.WithCredentialsFile(config.Get().GoogleApplicationCredentials))
	if err != nil {
		return 0, fmt.Errorf("error creating AlloyDB service: %w", err)
	}

	instanceName := fmt.Sprintf("projects/%s/locations/%s/clusters/%s/instances/%s", 
		config.Get().GCPProject, 
		config.Get().Region, 
		config.Get().ClusterName, 
		config.Get().InstanceName)
	instance, err := service.Projects.Locations.Clusters.Instances.Get(instanceName).Context(ctx).Do()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return 0, fmt.Errorf("timeout getting instance for total memory: %w", err)
		}
		return 0, fmt.Errorf("error getting instance for total memory: %w", err)
	}

	totalMemoryGB := float64(instance.MachineConfig.CpuCount) * 8

	return totalMemoryGB, nil
}

// UpdateReplicaCount updates the number of replicas in the read pool
func UpdateReplicaCount(ctx context.Context, count int) (*alloydb.Operation, error) {
	service, err := alloydb.NewService(ctx, option.WithCredentialsFile(config.Get().GoogleApplicationCredentials))
	if err != nil {
		return nil, fmt.Errorf("error creating AlloyDB service: %w", err)
	}

	instanceName := fmt.Sprintf("projects/%s/locations/%s/clusters/%s/instances/%s", 
		config.Get().GCPProject, 
		config.Get().Region, 
		config.Get().ClusterName, 
		config.Get().InstanceName)
	instance, err := service.Projects.Locations.Clusters.Instances.Get(instanceName).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("error getting instance: %w", err)
	}

	instance.ReadPoolConfig.NodeCount = int64(count)
	operation, err := service.Projects.Locations.Clusters.Instances.Patch(instanceName, instance).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("error initiating replica update operation: %w", err)
	}

	return operation, nil
}

// WaitForOperation waits for an AlloyDB operation to complete
func WaitForOperation(ctx context.Context, operation *alloydb.Operation) error {
	service, err := alloydb.NewService(ctx, option.WithCredentialsFile(config.Get().GoogleApplicationCredentials))
	if err != nil {
		return fmt.Errorf("error creating AlloyDB service: %w", err)
	}

	log.Info().
		Str("component", "alloydb").
		Str("action", "operation").
		Str("operationName", operation.Name).
		Msg("Operation in progress. Waiting...")

	startTime := time.Now()
	for {
		op, err := service.Projects.Locations.Operations.Get(operation.Name).Context(ctx).Do()
		if err != nil {
			return fmt.Errorf("error getting operation status: %w", err)
		}

		if op.Done {
			if op.Error != nil {
				return fmt.Errorf("operation failed: %s", op.Error.Message)
			}
			log.Info().
				Str("component", "alloydb").
				Str("action", "operation").
				Str("operationName", operation.Name).
				Str("duration", fmt.Sprintf("%.2fs", time.Since(startTime).Seconds())).
				Msg("Operation completed successfully")
			return nil
		}

		time.Sleep(10 * time.Second)
	}
}