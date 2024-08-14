package main

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	monitoring "cloud.google.com/go/monitoring/apiv3/v2"
	"cloud.google.com/go/monitoring/apiv3/v2/monitoringpb"
	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
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
	Evaluation                   int
	GCPProject                   string
	ClusterName                  string
	InstanceName                 string
	Region                       string
	MinReplicas                  int
	MaxReplicas                  int
	TimeoutSeconds               int
	LogLevel                     log.Level
}

var (
	cfg Config
)

func init() {
	// Configurar logrus
	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)

	if err := loadConfig(); err != nil {
		log.Fatal(err)
	}

	// Definir o nível de log após carregar a configuração
	log.SetLevel(cfg.LogLevel)
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

	cfg.TimeoutSeconds, err = parseIntConfig("TIMEOUT_SECONDS")
	if err != nil {
		return err
	}
	if cfg.TimeoutSeconds <= 0 {
		return fmt.Errorf("TIMEOUT_SECONDS deve ser maior que 0, valor atual: %d", cfg.TimeoutSeconds)
	}

	// Carregar e configurar o nível de log
	logLevelStr := os.Getenv("LOG_LEVEL")
	if logLevelStr == "" {
		logLevelStr = "info" // Nível padrão se não for especificado
	}
	logLevel, err := log.ParseLevel(strings.ToLower(logLevelStr))
	if err != nil {
		return fmt.Errorf("nível de log inválido '%s': %w", logLevelStr, err)
	}
	cfg.LogLevel = logLevel

	log.WithFields(log.Fields{
		"CPUThreshold":    cfg.CPUThreshold,
		"MemoryThreshold": cfg.MemoryThreshold,
		"CheckInterval":   cfg.CheckInterval,
		"Evaluation":      cfg.Evaluation,
		"MinReplicas":     cfg.MinReplicas,
		"MaxReplicas":     cfg.MaxReplicas,
		"TimeoutSeconds":  cfg.TimeoutSeconds,
		"LogLevel":        cfg.LogLevel.String(),
	}).Info("Configuração carregada com sucesso")

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
	log.WithFields(log.Fields{
		"appName":  AppName,
		"version":  runtime.Version(),
		"logLevel": cfg.LogLevel.String(),
	}).Info("Iniciando o aplicativo")

	ctx := context.Background()
	client, err := monitoring.NewMetricClient(ctx)
	if err != nil {
		log.WithError(err).Fatal("Falha ao criar cliente de métricas")
	}
	defer client.Close()

	var scaleUpCount, scaleDownCount int
	evaluationStart := time.Now()

	for {
		func() {
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.TimeoutSeconds)*time.Second)
			defer cancel()

			if err := checkMetrics(ctx, client, &scaleUpCount, &scaleDownCount); err != nil {
				if ctx.Err() == context.DeadlineExceeded {
					log.WithFields(log.Fields{
						"timeout": cfg.TimeoutSeconds,
						"error":   err,
					}).Error("Timeout ao verificar métricas")
				} else {
					log.WithError(err).Error("Erro ao verificar métricas")
				}
			}
		}()

		if time.Since(evaluationStart) >= time.Duration(cfg.Evaluation)*time.Second {
			if scaleUpCount > scaleDownCount && scaleUpCount > 0 {
				if err := scaleUp(context.Background()); err != nil {
					log.WithError(err).Error("Falha ao escalar para cima")
				} else {
					log.Info("Operação de escala de réplicas concluída com sucesso")
				}
			} else if scaleDownCount > scaleUpCount && scaleDownCount > 0 {
				if err := scaleDown(context.Background()); err != nil {
					log.WithError(err).Error("Falha ao escalar para baixo")
				} else {
					log.Info("Operação de redução de réplicas concluída com sucesso")
				}
			} else {
				log.Info("Nenhuma ação de escala necessária")
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
	log.WithField("proximaChecagem", nextCheck.Format("15:04:05")).Debug("Próxima checagem agendada")
	time.Sleep(time.Duration(duration) * time.Second)
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

	log.WithFields(log.Fields{
		"cluster":            cfg.ClusterName,
		"cpuUsagePercent":    cpuUsagePercent,
		"memoryUsagePercent": memoryUsagePercent,
	}).Info("Métricas atuais")

	currentCount, err := getReadPoolNodeCount(ctx)
	if err != nil {
		return err
	}

	if memoryUsagePercent > cfg.MemoryThreshold || cpuUsagePercent > cfg.CPUThreshold {
		if currentCount < cfg.MaxReplicas {
			log.WithFields(log.Fields{
				"cpuUsage":    cpuUsagePercent,
				"memoryUsage": memoryUsagePercent,
			}).Warn("Recursos insuficientes detectados, considerando escalar réplicas")
			*scaleUpCount++
			*scaleDownCount = 0
		} else {
			log.Warn("Recursos insuficientes detectados, mas o número máximo de réplicas já foi atingido")
		}
	} else if memoryUsagePercent < cfg.MemoryThreshold && cpuUsagePercent < cfg.CPUThreshold {
		if currentCount > cfg.MinReplicas {
			log.WithFields(log.Fields{
				"cpuUsage":    cpuUsagePercent,
				"memoryUsage": memoryUsagePercent,
			}).Warn("Recursos em excesso detectados, considerando reduzir o número de réplicas")
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
			if ctx.Err() == context.DeadlineExceeded {
				return 0, fmt.Errorf("timeout ao consultar métrica %s: %w", metricType, err)
			}
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
		if ctx.Err() == context.DeadlineExceeded {
			return 0, fmt.Errorf("timeout ao obter instância: %w", err)
		}
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
		if ctx.Err() == context.DeadlineExceeded {
			return 0, fmt.Errorf("timeout ao obter instância para memória total: %w", err)
		}
		return 0, fmt.Errorf("erro ao obter instância para memória total: %w", err)
	}

	totalMemoryGB := float64(instance.MachineConfig.CpuCount) * 8

	return totalMemoryGB, nil
}

func logNormalResources(count int) {
	log.WithField("replicaCount", count).Info("Recursos do AlloyDB dentro da normalidade")
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
		log.WithField("novaQuantidade", newCount).Info("Iniciando operação de escala")

		err = waitForOperation(ctx, operation)
		if err != nil {
			return fmt.Errorf("erro ao aguardar a conclusão da operação de escala: %w", err)
		}

		log.WithField("novaQuantidade", newCount).Info("Escalado com sucesso")
	} else {
		log.Warn("O número máximo de réplicas foi atingido. Não serão criadas novas réplicas")
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
		log.WithField("novaQuantidade", newCount).Info("Iniciando operação de redução")

		err = waitForOperation(ctx, operation)
		if err != nil {
			return fmt.Errorf("erro ao aguardar a conclusão da operação de redução: %w", err)
		}

		log.WithField("novaQuantidade", newCount).Info("Reduzido com sucesso")
	} else {
		log.Warn("O número mínimo de réplicas foi atingido. Não serão removidas réplicas")
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

	log.Info("Operação em andamento. Aguardando...")

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
			log.WithField("tempoTotal", time.Since(startTime).Round(time.Second)).Info("Operação concluída")
			return nil
		}

		time.Sleep(10 * time.Second)
	}
}
