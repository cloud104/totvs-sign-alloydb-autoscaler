from dotenv import load_dotenv, find_dotenv
import os
import subprocess
import requests
import time
import logging
import json

# Carregar variáveis de ambiente do arquivo .env se existir
dotenv_path = find_dotenv()
if dotenv_path:
    load_dotenv(dotenv_path)

# Configurar a autenticação do GCP
GOOGLE_APPLICATION_CREDENTIALS = os.getenv("GOOGLE_APPLICATION_CREDENTIALS")
os.environ["GOOGLE_APPLICATION_CREDENTIALS"] = GOOGLE_APPLICATION_CREDENTIALS

# Configurar o URL do Prometheus
PROMETHEUS_URL = os.getenv("PROMETHEUS_URL", "http://your-prometheus-server/api/v1/query")

# Autenticação básica do Prometheus
PROMETHEUS_USER = os.getenv("PROMETHEUS_USER")
PROMETHEUS_PASSWORD = os.getenv("PROMETHEUS_PASSWORD")

# Consultas de Prometheus para uso de memória e CPU
MEMORY_QUERY = 'max_over_time(stackdriver_alloydb_googleapis_com_instance_alloydb_googleapis_com_instance_memory_min_available_memory{instance_id="totvs-sign-alloydb-read"}[5m]) / 1024 / 1024 / 1024'
CPU_QUERY = 'max_over_time(stackdriver_alloydb_googleapis_com_instance_alloydb_googleapis_com_instance_cpu_average_utilization{instance_id="totvs-sign-alloydb-read"}[5m]) *100'

# Limiar de uso de memória e CPU configuráveis via variáveis de ambiente
DESIRE_FREE_MEMORY_THRESHOLD = float(os.getenv("DESIRE_FREE_MEMORY_THRESHOLD", "1"))  # Memória free desejada GB
CPU_THRESHOLD = float(os.getenv("CPU_THRESHOLD", "100"))       # Limiar de CPU alta

# Intervalo de verificação em segundos (pode ser configurado via variável de ambiente)
CHECK_INTERVAL = int(os.getenv("CHECK_INTERVAL", "300")) #5 minutos dura uma escala na GCP

# Cluster, região e limites de réplicas configuráveis via variáveis de ambiente
GCP_PROJECT = os.getenv("GCP_PROJECT", "your-gcp-project")
CLUSTER_NAME = os.getenv("CLUSTER_NAME", "your-cluster-name")
INSTANCE_NAME = os.getenv("INSTANCE_NAME", "your-instance-name")
REGION = os.getenv("REGION", "your-region-name")
MIN_REPLICAS = int(os.getenv("MIN_REPLICAS", "1"))
MAX_REPLICAS = int(os.getenv("MAX_REPLICAS", "1")) #regraAlloydb#Máximo_replicas:"20"

# Configuração de logging
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')

def query_prometheus(query):
    try:
        response = requests.get(PROMETHEUS_URL, params={'query': query}, auth=(PROMETHEUS_USER, PROMETHEUS_PASSWORD))
        response.raise_for_status()
        results = response.json()['data']['result']
        return float(results[0]['value'][1]) if results else None
    except requests.exceptions.RequestException as e:
        logging.error(f"Error querying Prometheus: {e}")
        return None

def get_read_pool_node_count():
    command = [
        "gcloud", "alloydb", "instances", "describe", INSTANCE_NAME,
        f"--project={GCP_PROJECT}",
        f"--cluster={CLUSTER_NAME}",
        f"--region={REGION}",
        "--format=json"
    ]
    result = subprocess.run(command, capture_output=True, text=True)
    if result.returncode != 0:
        logging.error(f"Error getting read pool node count: {result.stderr}")
        return None
    output = result.stdout.strip()
    try:
        data = json.loads(output)
        return data['readPoolConfig']['nodeCount']
    except (json.JSONDecodeError, KeyError) as e:
        logging.error(f"Error parsing JSON output: {e}")
        return None

def scale_up():
    current_count = get_read_pool_node_count()
    if current_count is not None and current_count < MAX_REPLICAS:
        new_count = current_count + 1
        command = [
            "gcloud", "alloydb", "instances", "update", INSTANCE_NAME,
            f"--read-pool-node-count={new_count}",
            f"--region={REGION}",
            f"--cluster={CLUSTER_NAME}",
            f"--project={GCP_PROJECT}"
        ]
        subprocess.run(command, check=True)
        logging.info(f"Scaled up to {new_count} replicas.")
    else:
        logging.info("Maximum number of replicas reached or error retrieving current count. No new replicas will be created.")

def scale_down():
    current_count = get_read_pool_node_count()
    if current_count is not None and current_count > MIN_REPLICAS:
        new_count = current_count - 1
        command = [
            "gcloud", "alloydb", "instances", "update", INSTANCE_NAME,
            f"--read-pool-node-count={new_count}",
            f"--region={REGION}",
            f"--cluster={CLUSTER_NAME}",
            f"--project={GCP_PROJECT}"
        ]
        subprocess.run(command, check=True)
        logging.info(f"Scaled down to {new_count} replicas.")
    else:
        logging.info("Cannot scale down below minimum replicas or error retrieving current count.")

# Função para verificar métricas e escalar/desescalar
def check_metrics():
    memory_free_gb = query_prometheus(MEMORY_QUERY)
    cpu_usage = query_prometheus(CPU_QUERY)

    if memory_free_gb is not None and memory_free_gb < DESIRE_FREE_MEMORY_THRESHOLD:
        logging.info("Low memory free detected, scaling up.")
        scale_up()
        return
    
    if cpu_usage is not None and cpu_usage > CPU_THRESHOLD:
        logging.info("High CPU usage detected, scaling up.")
        scale_up()
        return

    if memory_free_gb is not None and memory_free_gb > DESIRE_FREE_MEMORY_THRESHOLD:
        logging.info("High memory free detected, scaling down.")
        scale_down()
        return
    
    if cpu_usage is not None and cpu_usage < CPU_THRESHOLD:
        logging.info("Low CPU usage detected, scaling down.")
        scale_down()
        return

# Executar a verificação de métricas periodicamente
if __name__ == "__main__":
    while True:
        check_metrics()
        time.sleep(CHECK_INTERVAL)  # Verificar com base no intervalo configurado