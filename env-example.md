# Google Cloud Platform credentials
GOOGLE_APPLICATION_CREDENTIALS=key.json
GCP_PROJECT=my-project
CLUSTER_NAME=my-cluster
INSTANCE_NAME=my-instance
REGION=region         # e.g. us-central1

LOG_LEVEL=DEBUG
CPU_THRESHOLD=90 # Escala AlloyDB com CPU acima de 90%.
MEMORY_THRESHOLD=90 # Escala AlloyDB com memoria acima de 90%.
CHECK_INTERVAL=10 # Verifica a cada 10 segundos
EVALUATION=60 # Avalia numa janela de 2 minutos todas as verificações de 10 segundos para daí sim escalar ou desescalar.
MIN_REPLICAS=1 # Minimo de replicas
MAX_REPLICAS=2 # Maximo de replias
TIMEOUT_SECONDS=10 # Timeout da API da GCP em segundos




ROADMAP:


Usar a propria gcp pra fornecer a metrica. V

usar o sdk direto no codigo. V

remover o cronometro e printar a hora exata da proxima execucao. V

carregar o ascii_art no binario principal ao inves de depender do arquivo. V

Por quanto tempo ele esta nessa condicao para poder aumentar ou diminuir replicas?   V
Parametro personalizavel! V

Remover Bobagem do ascii_art.




Mudar o log para Json (melhor visualizacao em datadog e elasticsearch)

colocar o log detalhado como debug.

colocar no /metrics as metricas que foram buscadas na gcp.

log info: printar o nome do Cluster em todas as linhas de log.

cada log com spanID unico

exemplo: Cluster: teste: proxima checagem

colocar trace_id para correlacionar logs entre controller e cluster/namespace.

criar metrica, service_state_up com todas as variavies de ambiente printadas.

#EXEMPLO de LOG JSON:

{
	"context":"WorkerController",
	"level":"info",
	"message":"escalando cluster..",
	"span_id":"f85ee78f-9bd7-4e41-9409-bbc25a753d72",
	
	"trace_id":"717eb53e-e7c6-49eb-bc75-850be8fff0af", // transação
	
	"value": {
		
		"object":"cluster_name",
		"namespace":"namespace_name",
		"currentCpu":"20%",
		"currentMem":"25%",
		"targetCpu":"18%",
		"targetMem":"50%"
	}
}
{
	"context":"WithInputEvent",
	"level":"info",
	"message":"event was parsed",
	"span_id":"ec9089ef-d7dc-412a-9d65-b7587edaf44e",
	"trace_id":"fcb2b22d-9cf0-4879-ae5d-7021e61b1e0d",
	"value":{
		"context":"WithInputEvent",
		"event":{
			"exchange":"supply.foundation-exchange",
			"type":"JobTarefaExecutadaEvent"
		},
		"id":"2352ad40-9c03-4039-b774-9a6d69404677",
		"tenant":"71e8f160-d9d4-4a2d-8707-781443ac726c",
		"transaction":"fcb2b22d-9cf0-4879-ae5d-7021e61b1e0d"
	}
}