# Google Cloud Platform credentials
GOOGLE_APPLICATION_CREDENTIALS=key.json
GCP_PROJECT=my-project
CLUSTER_NAME=my-cluster
INSTANCE_NAME=my-instance
REGION=region         # e.g. us-central1

LOG_LEVEL=INFO

CPU_THRESHOLD=90 # Escala AlloyDB com CPU acima de 90%.
MEMORY_THRESHOLD=90 # Escala AlloyDB com memoria acima de 90%.

CHECK_INTERVAL=30 # Verifica a cada 10 segundos
EVALUATION=120 # Avalia numa janela de 2 minutos todas as verificações de 10 segundos para daí sim escalar ou desescalar.
MIN_REPLICAS=1 # Minimo de replicas
MAX_REPLICAS=2 # Máximo de replicas