package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"runtime"
	"strconv"
	"time"

	monitoring "cloud.google.com/go/monitoring/apiv3/v2"
	"cloud.google.com/go/monitoring/apiv3/v2/monitoringpb"
	"github.com/common-nighthawk/go-figure"
	"github.com/joho/godotenv"
	"google.golang.org/api/alloydb/v1"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Config armazena todas as configurações do aplicativo
type Config struct {
	GoogleApplicationCredentials string
	MemoryMetric                 string
	CPUMetric                    string
	CPUThreshold                 float64
	MemoryThreshold              float64
	CheckInterval                int
	EvaluationPeriod             int
	GCPProject                   string
	ClusterName                  string
	InstanceName                 string
	Region                       string
	MinReplicas                  int
	MaxReplicas                  int
}

var (
	cfg Config
)

func init() {
	if err := loadConfig(); err != nil {
		log.Fatal(err)
	}

	// Configurar o formato de log para usar o padrão brasileiro de data
	log.SetFlags(0)
	log.SetOutput(new(LogWriter))
}

type LogWriter struct{}

func (writer LogWriter) Write(bytes []byte) (int, error) {
	return fmt.Print(time.Now().Format("02/01/2006 15:04:05") + " " + string(bytes))
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

	cfg.EvaluationPeriod, err = parseIntConfig("EVALUATION_PERIOD")
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

const AppName = "AlloyDB Autoscaler"

func main() {
	Banner := figure.NewFigure("TOTVS APPS", "small", true)
	Banner.Print()
	fmt.Println()
	fmt.Printf("%s (%s): Iniciando o aplicativo...\n\n", AppName, runtime.Version())

	if err := loadConfig(); err != nil {
		log.Fatalf("Erro ao carregar configuração: %v", err)
	}

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
			log.Println(err)
		}

		if time.Since(evaluationStart) >= time.Duration(cfg.EvaluationPeriod)*time.Second {
			if scaleUpCount > scaleDownCount && scaleUpCount > 0 {
				if err := scaleUp(ctx); err != nil {
					log.Println(err)
				} else {
					log.Println("INFO: Operação de escala de réplicas concluída com sucesso.")
				}
			} else if scaleDownCount > scaleUpCount && scaleDownCount > 0 {
				if err := scaleDown(ctx); err != nil {
					log.Println(err)
				} else {
					log.Println("INFO: Operação de redução de réplicas concluída com sucesso.")
				}
			} else {
				log.Println("INFO: Nenhuma ação de escala necessária.")
			}

			scaleUpCount = 0
			scaleDownCount = 0
			evaluationStart = time.Now()
		}

		logTimer(cfg.CheckInterval)
	}
}

func logTimer(duration int) {
	nextCheck := time.Now().Add(time.Duration(duration) * time.Second)
	log.Printf("INFO: Próxima checagem: %s", nextCheck.Format("15:04:05"))
	time.Sleep(time.Duration(duration) * time.Second)
	//log.Println("INFO: Iniciando nova checagem")
	fmt.Println() // Adiciona uma linha em branco
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

	log.Printf("INFO: Cluster: %s, Uso de CPU: %.2f%%, Uso de Memória: %.2f%%",
		cfg.ClusterName, cpuUsagePercent, memoryUsagePercent)

	currentCount, err := getReadPoolNodeCount(ctx)
	if err != nil {
		return err
	}

	if memoryUsagePercent > cfg.MemoryThreshold || cpuUsagePercent > cfg.CPUThreshold {
		if currentCount < cfg.MaxReplicas {
			log.Println("INFO: \033[31mRecursos insuficientes detectados, considerando escalar réplicas.\033[0m")
			*scaleUpCount++
			*scaleDownCount = 0
		} else {
			log.Println("INFO: Recursos insuficientes detectados, mas o número máximo de réplicas já foi atingido.")
		}
	} else if memoryUsagePercent < cfg.MemoryThreshold && cpuUsagePercent < cfg.CPUThreshold {
		if currentCount > cfg.MinReplicas {
			log.Println("INFO: \033[34mRecursos em excesso detectados, considerando reduzir o número de réplicas.\033[0m")
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
	var lastValue float64
	for {
		resp, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return 0, fmt.Errorf("erro ao iterar sobre as séries temporais: %w", err)
		}
		if len(resp.Points) > 0 {
			switch v := resp.Points[0].Value.Value.(type) {
			case *monitoringpb.TypedValue_DoubleValue:
				lastValue = v.DoubleValue
			case *monitoringpb.TypedValue_Int64Value:
				lastValue = float64(v.Int64Value)
			default:
				return 0, fmt.Errorf("tipo de valor não suportado: %T", v)
			}
		}
	}

	return lastValue, nil
}

func getReadPoolNodeCount(ctx context.Context) (int, error) {
	service, err := alloydb.NewService(ctx, option.WithCredentialsFile(cfg.GoogleApplicationCredentials))
	if err != nil {
		return 0, fmt.Errorf("erro ao criar serviço AlloyDB: %w", err)
	}

	instanceName := fmt.Sprintf("projects/%s/locations/%s/clusters/%s/instances/%s", cfg.GCPProject, cfg.Region, cfg.ClusterName, cfg.InstanceName)
	instance, err := service.Projects.Locations.Clusters.Instances.Get(instanceName).Context(ctx).Do()
	if err != nil {
		return 0, fmt.Errorf("erro ao obter instância: %w", err)
	}

	return int(instance.ReadPoolConfig.NodeCount), nil
}

func getTotalMemory(ctx context.Context) (float64, error) {
	service, err := alloydb.NewService(ctx, option.WithCredentialsFile(cfg.GoogleApplicationCredentials))
	if err != nil {
		return 0, fmt.Errorf("erro ao criar serviço AlloyDB: %w", err)
	}

	instanceName := fmt.Sprintf("projects/%s/locations/%s/clusters/%s/instances/%s", cfg.GCPProject, cfg.Region, cfg.ClusterName, cfg.InstanceName)
	instance, err := service.Projects.Locations.Clusters.Instances.Get(instanceName).Context(ctx).Do()
	if err != nil {
		return 0, fmt.Errorf("erro ao obter instância: %w", err)
	}

	// A memória é geralmente fornecida em GB
	totalMemoryGB := float64(instance.MachineConfig.CpuCount) * 8 // Assumindo 16GB por vCPU, ajuste conforme necessário

	return totalMemoryGB, nil
}

func logNormalResources(count int) {
	log.Printf("INFO: \033[32mRecursos do AlloyDB dentro da normalidade com %d réplicas.\033[0m", count)
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
		log.Printf("INFO: Iniciando operação de escala para %d réplicas.", newCount)

		err = waitForOperation(ctx, operation)
		if err != nil {
			return fmt.Errorf("erro ao aguardar a conclusão da operação de escala: %w", err)
		}

		fmt.Println() // Adiciona uma quebra de linha após a conclusão da operação
		log.Printf("INFO: Escalado com sucesso para %d réplicas.", newCount)
	} else {
		log.Println("INFO: O número máximo de réplicas foi atingido. Não serão criadas novas réplicas.")
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
		log.Printf("INFO: Iniciando operação de redução para %d réplicas.", newCount)

		err = waitForOperation(ctx, operation)
		if err != nil {
			return fmt.Errorf("erro ao aguardar a conclusão da operação de redução: %w", err)
		}

		fmt.Println() // Adiciona uma quebra de linha após a conclusão da operação
		log.Printf("INFO: Reduzido com sucesso para %d réplicas.", newCount)
	} else {
		log.Println("INFO: O número mínimo de réplicas foi atingido. Não serão removidas réplicas.")
	}
	return nil
}

func updateReplicaCount(ctx context.Context, count int) (*alloydb.Operation, error) {
	service, err := alloydb.NewService(ctx, option.WithCredentialsFile(cfg.GoogleApplicationCredentials))
	if err != nil {
		return nil, fmt.Errorf("erro ao criar serviço AlloyDB: %w", err)
	}

	instanceName := fmt.Sprintf("projects/%s/locations/%s/clusters/%s/instances/%s", cfg.GCPProject, cfg.Region, cfg.ClusterName, cfg.InstanceName)
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

func waitForOperation(ctx context.Context, operation *alloydb.Operation) error {
	service, err := alloydb.NewService(ctx, option.WithCredentialsFile(cfg.GoogleApplicationCredentials))
	if err != nil {
		return fmt.Errorf("erro ao criar serviço AlloyDB: %w", err)
	}

	log.Println("INFO: Operação em andamento. Aguardando...")

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
			log.Printf("INFO: Operação concluída. Tempo total: %v", time.Since(startTime).Round(time.Second))
			return nil
		}

		// Imprimir um ponto a cada 10 segundos para indicar que a operação ainda está em andamento
		fmt.Print(".")
		time.Sleep(10 * time.Second)
	}
}
