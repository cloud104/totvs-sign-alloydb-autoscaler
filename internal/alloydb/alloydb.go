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

// GetReadPoolNodeCount retorna o número atual de nós no pool de leitura
func GetReadPoolNodeCount(ctx context.Context) (int, error) {
	service, err := alloydb.NewService(ctx, option.WithCredentialsFile(config.Get().GoogleApplicationCredentials))
	if err != nil {
		return 0, fmt.Errorf("erro ao criar serviço AlloyDB: %w", err)
	}

	instanceName := fmt.Sprintf("projects/%s/locations/%s/clusters/%s/instances/%s", 
		config.Get().GCPProject, 
		config.Get().Region, 
		config.Get().ClusterName, 
		config.Get().InstanceName)
	instance, err := service.Projects.Locations.Clusters.Instances.Get(instanceName).Context(ctx).Do()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return 0, fmt.Errorf("timeout ao obter instância: %w", err)
		}
		return 0, fmt.Errorf("erro ao obter instância: %w", err)
	}

	return int(instance.ReadPoolConfig.NodeCount), nil
}

// GetTotalMemory retorna a memória total da instância em GB
func GetTotalMemory(ctx context.Context) (float64, error) {
	service, err := alloydb.NewService(ctx, option.WithCredentialsFile(config.Get().GoogleApplicationCredentials))
	if err != nil {
		return 0, fmt.Errorf("erro ao criar serviço AlloyDB: %w", err)
	}

	instanceName := fmt.Sprintf("projects/%s/locations/%s/clusters/%s/instances/%s", 
		config.Get().GCPProject, 
		config.Get().Region, 
		config.Get().ClusterName, 
		config.Get().InstanceName)
	instance, err := service.Projects.Locations.Clusters.Instances.Get(instanceName).Context(ctx).Do()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return 0, fmt.Errorf("timeout ao obter instância para memória total: %w", err)
		}
		return 0, fmt.Errorf("erro ao obter instância para memória total: %w", err)
	}

	totalMemoryGB := float64(instance.MachineConfig.CpuCount) * 8

	return totalMemoryGB, nil
}

// UpdateReplicaCount atualiza o número de réplicas no pool de leitura
func UpdateReplicaCount(ctx context.Context, count int) (*alloydb.Operation, error) {
	service, err := alloydb.NewService(ctx, option.WithCredentialsFile(config.Get().GoogleApplicationCredentials))
	if err != nil {
		return nil, fmt.Errorf("erro ao criar serviço AlloyDB: %w", err)
	}

	instanceName := fmt.Sprintf("projects/%s/locations/%s/clusters/%s/instances/%s", 
		config.Get().GCPProject, 
		config.Get().Region, 
		config.Get().ClusterName, 
		config.Get().InstanceName)
	instance, err := service.Projects.Locations.Clusters.Instances.Get(instanceName).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("erro ao obter instância: %w", err)
	}

	instance.ReadPoolConfig.NodeCount = int64(count)
	operation, err := service.Projects.Locations.Clusters.Instances.Patch(instanceName, instance).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("erro ao iniciar operação de atualização de réplicas: %w", err)
	}

	return operation, nil
}

// WaitForOperation aguarda a conclusão de uma operação do AlloyDB
func WaitForOperation(ctx context.Context, operation *alloydb.Operation) error {
	service, err := alloydb.NewService(ctx, option.WithCredentialsFile(config.Get().GoogleApplicationCredentials))
	if err != nil {
		return fmt.Errorf("erro ao criar serviço AlloyDB: %w", err)
	}

	log.Info().Msg("Operação em andamento. Aguardando...")

	startTime := time.Now()
	for {
		op, err := service.Projects.Locations.Operations.Get(operation.Name).Context(ctx).Do()
		if err != nil {
			return fmt.Errorf("erro ao obter status da operação: %w", err)
		}

		if op.Done {
			if op.Error != nil {
				return fmt.Errorf("operação falhou: %s", op.Error.Message)
			}
			log.Info().Dur("tempoTotal", time.Since(startTime).Round(time.Second)).Msg("Operação concluída")
			return nil
		}

		time.Sleep(10 * time.Second)
	}
}
