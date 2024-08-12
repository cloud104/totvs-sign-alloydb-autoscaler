package main

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"time"

	monitoring "cloud.google.com/go/monitoring/apiv3/v2"
	"cloud.google.com/go/monitoring/apiv3/v2/monitoringpb"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/alloydb/v1"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Config struct {
	GoogleApplicationCredentials string
	MemoryMetric                 string
	CPUMetric                    string
	CPUThreshold                 float64
	MemoryThreshold              float64
	CheckInterval                int
	Evaluation                   int
	GCPProject                   string
	ClusterName                  string
	InstanceName                 string
	Region                       string
	MinReplicas                  int
	MaxReplicas                  int
	LogLevel                     string
}

var (
	cfg Config
	log = logrus.New()
)

const AppName = "AlloyDB Autoscaler"

func init() {
	if err := loadConfig(); err != nil {
		log.Fatal(err)
	}

	log.SetFormatter(&logrus.TextFormatter{
		TimestampFormat: "02/01/2006 15:04:05",
		FullTimestamp:   true,
	})

	level, err := logrus.ParseLevel(cfg.LogLevel)
	if err != nil {
		log.Fatal(err)
	}
	log.SetLevel(level)
}

func loadConfig() error {
	if err := godotenv.Load("/app/.env"); err != nil {
		return fmt.Errorf("erro ao carregar arquivo .env: %w", err)
	}

	var err error
	cfg = Config{
		GoogleApplicationCredentials: os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"),
		MemoryMetric:                 "alloydb.googleapis.com/instance/memory/min_available_memory",
		CPUMetric:                    "alloydb.googleapis.com/instance/cpu/average_utilization",
		GCPProject:                   os.Getenv("GCP_PROJECT"),
		ClusterName:                  os.Getenv("CLUSTER_NAME"),
		InstanceName:                 os.Getenv("INSTANCE_NAME"),
		Region:                       os.Getenv("REGION"),
		LogLevel:                     os.Getenv("LOG_LEVEL"),
	}

	cfg.CPUThreshold, err = parseFloatConfig("CPU_THRESHOLD")
	if err != nil {
		return err
	}

	cfg.MemoryThreshold, err = parseFloatConfig("MEMORY_THRESHOLD")
	if err != nil {
		return err
	}

	cfg.CheckInterval, err = parseIntConfig("CHECK_INTERVAL")
	if err != nil {
		return err
	}

	cfg.Evaluation, err = parseIntConfig("EVALUATION")
	if err != nil {
		return err
	}

	cfg.MinReplicas, err = parseIntConfig("MIN_REPLICAS")
	if err != nil {
		return err
	}
	if cfg.MinReplicas < 1 {
		return fmt.Errorf("MIN_REPLICAS deve ser pelo menos 1, valor atual: %d", cfg.MinReplicas)
	}

	cfg.MaxReplicas, err = parseIntConfig("MAX_REPLICAS")
	if err != nil {
		return err
	}
	if cfg.MaxReplicas > 20 {
		return fmt.Errorf("MAX_REPLICAS não pode exceder 20, valor atual: %d", cfg.MaxReplicas)
	}

	if cfg.MinReplicas > cfg.MaxReplicas {
		return fmt.Errorf("MIN_REPLICAS (%d) não pode ser maior que MAX_REPLICAS (%d)", cfg.MinReplicas, cfg.MaxReplicas)
	}

	return nil
}

func parseFloatConfig(key string) (float64, error) {
	value := os.Getenv(key)
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, fmt.Errorf("revise %s: o valor '%s' é inválido. Certifique-se de que o valor seja um número válido sem letras ou caracteres especiais", key, value)
	}
	return parsed, nil
}

func parseIntConfig(key string) (int, error) {
	value := os.Getenv(key)
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("revise %s: o valor '%s' é inválido. Certifique-se de que o valor seja um número inteiro válido sem letras ou caracteres especiais", key, value)
	}
	return parsed, nil
}

func main() {
	fmt.Println()
	log.Infof("%s (%s): Iniciando o aplicativo...", AppName, runtime.Version())

	ctx := context.Background()
	client, err := monitoring.NewMetricClient(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	var scaleUpCount, scaleDownCount int
	evaluationStart := time.Now()

	for {
		if err := checkMetrics(ctx, client, &scaleUpCount, &scaleDownCount); err != nil {
			log.Errorf("Erro ao verificar métricas: %v", err)
		}

		if time.Since(evaluationStart) >= time.Duration(cfg.Evaluation)*time.Second {
			if scaleUpCount > scaleDownCount && scaleUpCount > 0 {
				if err := scaleUp(ctx); err != nil {
					log.Errorf("Erro ao escalar: %v", err)
				} else {
					log.Infof("Operação de escala de réplicas concluída com sucesso.")
				}
			} else if scaleDownCount > scaleUpCount && scaleDownCount > 0 {
				if err := scaleDown(ctx); err != nil {
					log.Errorf("Erro ao reduzir réplicas: %v", err)
				} else {
					log.Infof("Operação de redução de réplicas concluída com sucesso.")
				}
			} else {
				log.Infof("Nenhuma ação de escala necessária.")
			}

			scaleUpCount = 0
			scaleDownCount = 0
			evaluationStart = time.Now()
		}

		logTimer(cfg.CheckInterval)
	}
}

func retry(attempts int, sleep time.Duration, f func() error) error {
	var err error
	for i := 0; i < attempts; i++ {
		if err = f(); err == nil {
			return nil
		}
		time.Sleep(sleep)
		sleep *= 2
	}
	return fmt.Errorf("após %d tentativas, última erro: %v", attempts, err)
}

func retryAlloyDBOperation(ctx context.Context, operation func(*alloydb.Service) error) error {
	return retry(3, time.Second, func() error {
		service, err := alloydb.NewService(ctx, option.WithCredentialsFile(cfg.GoogleApplicationCredentials))
		if err != nil {
			return fmt.Errorf("erro ao criar serviço AlloyDB: %w", err)
		}
		return operation(service)
	})
}

func getInstanceWithRetry(ctx context.Context) (*alloydb.Instance, error) {
	var instance *alloydb.Instance
	err := retryAlloyDBOperation(ctx, func(service *alloydb.Service) error {
		instanceName := fmt.Sprintf("projects/%s/locations/%s/clusters/%s/instances/%s", cfg.GCPProject, cfg.Region, cfg.ClusterName, cfg.InstanceName)
		var err error
		instance, err = service.Projects.Locations.Clusters.Instances.Get(instanceName).Context(ctx).Do()
		return err
	})
	return instance, err
}

func logTimer(duration int) {
	nextCheck := time.Now().Add(time.Duration(duration) * time.Second)
	log.Debugf("Próxima checagem: %s", nextCheck.Format("15:04:05"))
	time.Sleep(time.Duration(duration) * time.Second)
	fmt.Println()
}

func checkMetrics(ctx context.Context, client *monitoring.MetricClient, scaleUpCount, scaleDownCount *int) error {
	memoryFreeBytes, err := queryMetric(ctx, client, cfg.MemoryMetric)
	if err != nil {
		return fmt.Errorf("erro ao consultar memória livre: %w", err)
	}

	cpuUsage, err := queryMetric(ctx, client, cfg.CPUMetric)
	if err != nil {
		return fmt.Errorf("erro ao consultar uso de CPU: %w", err)
	}

	totalMemoryGB, err := getTotalMemory(ctx)
	if err != nil {
		return fmt.Errorf("erro ao obter memória total: %w", err)
	}

	memoryFreeGB := memoryFreeBytes / (1024 * 1024 * 1024)
	memoryUsedGB := totalMemoryGB - memoryFreeGB
	memoryUsagePercent := (memoryUsedGB / totalMemoryGB) * 100

	cpuUsagePercent := cpuUsage * 100

	log.Debugf("Cluster: %s, Uso de CPU: %.2f%%, Uso de Memória: %.2f%%",
		cfg.ClusterName, cpuUsagePercent, memoryUsagePercent)

	currentCount, err := getReadPoolNodeCount(ctx)
	if err != nil {
		return err
	}

	if memoryUsagePercent > cfg.MemoryThreshold || cpuUsagePercent > cfg.CPUThreshold {
		if currentCount < cfg.MaxReplicas {
			log.Infof("Recursos insuficientes detectados, considerando escalar réplicas.")
			*scaleUpCount++
			*scaleDownCount = 0
		} else {
			log.Debugf("Recursos insuficientes detectados, mas o número máximo de réplicas já foi atingido.")
		}
	} else if memoryUsagePercent < cfg.MemoryThreshold && cpuUsagePercent < cfg.CPUThreshold {
		if currentCount > cfg.MinReplicas {
			log.Infof("Recursos em excesso detectados, considerando reduzir o número de réplicas.")
			*scaleDownCount++
			*scaleUpCount = 0
		} else {
			logNormalResources(currentCount)
		}
	} else {
		logNormalResources(currentCount)
		*scaleUpCount = 0
		*scaleDownCount = 0
	}

	return nil
}

func queryMetric(ctx context.Context, client *monitoring.MetricClient, metricType string) (float64, error) {
	var lastValue float64
	err := retry(3, time.Second, func() error {
		now := time.Now()
		startTime := now.Add(-5 * time.Minute)

		req := &monitoringpb.ListTimeSeriesRequest{
			Name:   fmt.Sprintf("projects/%s", cfg.GCPProject),
			Filter: fmt.Sprintf(`metric.type = "%s" AND resource.labels.instance_id = "%s"`, metricType, cfg.InstanceName),
			Interval: &monitoringpb.TimeInterval{
				StartTime: timestamppb.New(startTime),
				EndTime:   timestamppb.New(now),
			},
			View: monitoringpb.ListTimeSeriesRequest_FULL,
		}

		it := client.ListTimeSeries(ctx, req)
		for {
			resp, err := it.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return fmt.Errorf("erro ao iterar sobre as séries temporais: %w", err)
			}
			if len(resp.Points) > 0 {
				switch v := resp.Points[0].Value.Value.(type) {
				case *monitoringpb.TypedValue_DoubleValue:
					lastValue = v.DoubleValue
				case *monitoringpb.TypedValue_Int64Value:
					lastValue = float64(v.Int64Value)
				default:
					return fmt.Errorf("tipo de valor não suportado: %T", v)
				}
			}
		}
		return nil
	})
	return lastValue, err
}

func getReadPoolNodeCount(ctx context.Context) (int, error) {
	instance, err := getInstanceWithRetry(ctx)
	if err != nil {
		return 0, err
	}
	return int(instance.ReadPoolConfig.NodeCount), nil
}

func getTotalMemory(ctx context.Context) (float64, error) {
	instance, err := getInstanceWithRetry(ctx)
	if err != nil {
		return 0, err
	}
	return float64(instance.MachineConfig.CpuCount) * 8, nil
}

func logNormalResources(count int) {
	log.Infof("Recursos do AlloyDB dentro da normalidade com %d réplicas.", count)
}

func scaleUp(ctx context.Context) error {
	currentCount, err := getReadPoolNodeCount(ctx)
	if err != nil {
		return err
	}

	if currentCount < cfg.MaxReplicas {
		newCount := currentCount + 1
		operation, err := updateReplicaCount(ctx, newCount)
		if err != nil {
			return err
		}
		log.Infof("Iniciando operação de escala para %d réplicas.", newCount)

		err = waitForOperation(ctx, operation)
		if err != nil {
			return fmt.Errorf("erro ao aguardar a conclusão da operação de escala: %w", err)
		}

		fmt.Println()
		log.Infof("Escalado com sucesso para %d réplicas.", newCount)
	} else {
		log.Infof("O número máximo de réplicas foi atingido. Não serão criadas novas réplicas.")
	}
	return nil
}

func scaleDown(ctx context.Context) error {
	currentCount, err := getReadPoolNodeCount(ctx)
	if err != nil {
		return err
	}

	if currentCount > cfg.MinReplicas {
		newCount := currentCount - 1
		operation, err := updateReplicaCount(ctx, newCount)
		if err != nil {
			return err
		}
		log.Infof("Iniciando operação de redução para %d réplicas.", newCount)
		err = waitForOperation(ctx, operation)
		if err != nil {
			return fmt.Errorf("erro ao aguardar a conclusão da operação de redução: %w", err)
		}

		fmt.Println()
		log.Infof("Reduzido com sucesso para %d réplicas.", newCount)
	} else {
		log.Infof("O número mínimo de réplicas foi atingido. Não serão removidas réplicas.")
	}
	return nil
}

func updateReplicaCount(ctx context.Context, count int) (*alloydb.Operation, error) {
	var operation *alloydb.Operation
	err := retryAlloyDBOperation(ctx, func(service *alloydb.Service) error {
		instanceName := fmt.Sprintf("projects/%s/locations/%s/clusters/%s/instances/%s", cfg.GCPProject, cfg.Region, cfg.ClusterName, cfg.InstanceName)
		instance, err := service.Projects.Locations.Clusters.Instances.Get(instanceName).Context(ctx).Do()
		if err != nil {
			return fmt.Errorf("erro ao obter instância: %w", err)
		}

		instance.ReadPoolConfig.NodeCount = int64(count)
		operation, err = service.Projects.Locations.Clusters.Instances.Patch(instanceName, instance).Context(ctx).Do()
		return err
	})
	return operation, err
}

func waitForOperation(ctx context.Context, operation *alloydb.Operation) error {
	log.Infof("Operação em andamento. Aguardando...")
	startTime := time.Now()

	return retryAlloyDBOperation(ctx, func(service *alloydb.Service) error {
		op, err := service.Projects.Locations.Operations.Get(operation.Name).Context(ctx).Do()
		if err != nil {
			return fmt.Errorf("erro ao obter status da operação: %w", err)
		}

		if op.Done {
			if op.Error != nil {
				return fmt.Errorf("operação falhou: %s", op.Error.Message)
			}
			log.Infof("Operação concluída. Tempo total: %v", time.Since(startTime).Round(time.Second))
			return nil
		}

		fmt.Print(".")
		return fmt.Errorf("operação ainda em andamento")
	})
}
