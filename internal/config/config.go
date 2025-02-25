package config

import (
	"fmt"
	"os"
	"strconv"
	"sync"

	"github.com/heraque/alloydb-autoscaler/tree/main/internal/log"
	"github.com/joho/godotenv"
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
	LogLevel                     string
}

var (
	cfg  Config
	once sync.Once
)

func init() {
	if err := Load(); err != nil {
		log.Fatal().Err(err).Msg("Falha ao carregar configuração")
	}

	log.Initialize()
}

// Get retorna a configuração atual
func Get() Config {
	return cfg
}

// Load carrega as configurações do ambiente
func Load() error {
	var err error
	once.Do(func() {
		err = loadConfig()
	})
	return err
}

func loadConfig() error {
	// Tenta carregar o .env, mas não falha se não existir
	_ = godotenv.Load("/app/.env")

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

	cfg.TimeoutSeconds, err = parseIntConfig("TIMEOUT_SECONDS")
	if err != nil {
		return err
	}
	if cfg.TimeoutSeconds <= 0 {
		return fmt.Errorf("TIMEOUT_SECONDS deve ser maior que 0, valor atual: %d", cfg.TimeoutSeconds)
	}

	log.Debug().
		Float64("CPUThreshold", cfg.CPUThreshold).
		Float64("MemoryThreshold", cfg.MemoryThreshold).
		Int("CheckInterval", cfg.CheckInterval).
		Int("Evaluation", cfg.Evaluation).
		Int("MinReplicas", cfg.MinReplicas).
		Int("MaxReplicas", cfg.MaxReplicas).
		Int("TimeoutSeconds", cfg.TimeoutSeconds).
		Msg("Configuração carregada com sucesso")

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
