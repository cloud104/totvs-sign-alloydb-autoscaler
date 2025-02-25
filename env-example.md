#Google Cloud Platform credentials
GOOGLE_APPLICATION_CREDENTIALS=key.json # Arquivo de credenciais da GCP
GCP_PROJECT= # Nome do projeto
CLUSTER_NAME= # Nome do cluster AlloyDB
INSTANCE_NAME= # Nome da instância AlloyDB
REGION= # Região do AlloyDB

LOG_LEVEL=info # Nível de log

CPU_THRESHOLD=90 # Escala AlloyDB com CPU acima de 90%.

MEMORY_THRESHOLD=90 # Escala AlloyDB com memoria acima de 90%.

CONNECTION_THRESHOLD= (Em construção...) # Escala AlloyDB com conexões acima de 90%.

ESCALAR_THRESHOLD= (Em construção...) # Quantas réplicas serão adicionadas ou removidas por verificação.

CHECK_INTERVAL=60 # Verifica a cada 60 segundos

EVALUATION=120 # Avalia numa janela de 120 segundos se todas as verificações de 60 segundos deram positivo ou negativo relacionado aos tressholds.

MIN_REPLICAS=1 # Minimo de replicas

MAX_REPLICAS=2 # Máximo de réplicas

TIMEOUT_SECONDS=10 # Timeout da API da GCP em segundos