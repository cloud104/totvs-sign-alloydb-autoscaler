AlloyDB Autoscaler
=====================

Overview
--------

Este aplicativo é um autoscaler para o AlloyDB que monitora a utilização de recursos (memória e CPU) e ajusta o número de réplicas com base nos limiares configurados.

Configuração
------------

O aplicativo lê as configurações do arquivo `.env` e as armazena em uma estrutura `Config`. As configurações incluem:

* `GoogleApplicationCredentials`: credenciais de aplicação do Google
* `MemoryMetric`: métrica de memória a ser monitorada
* `CPUMetric`: métrica de CPU a ser monitorada
* `DesireFreeMemoryThreshold`: limiar de memória livre desejado
* `CPUThreshold`: limiar de uso de CPU
* `CheckInterval`: intervalo de tempo entre as checagens
* `GCPProject`: projeto do Google Cloud Platform
* `ClusterName`: nome do cluster AlloyDB
* `InstanceName`: nome da instância AlloyDB
* `Region`: região do cluster AlloyDB
* `MinReplicas`: número mínimo de réplicas
* `MaxReplicas`: número máximo de réplicas

Funções
--------

### `checkMetrics`

Verifica as métricas de memória e CPU e ajusta o número de réplicas com base nos limiares configurados.

### `queryMetric`

Consulta a métrica especificada e retorna o valor mais recente.

### `getReadPoolNodeCount`

Obtém o número de nós do pool de leitura da instância AlloyDB.

### `scaleUp`

Aumenta o número de réplicas se a utilização de recursos for alta.

### `scaleDown`

Diminui o número de réplicas se a utilização de recursos for baixa.

### `updateReplicaCount`

Atualiza o número de réplicas da instância AlloyDB.

### `logTimer`

Imprime o tempo de início da próxima checagem e dorme pelo intervalo de tempo configurado.

 Estruturas de dados
-------------------

### `Data`

Estrutura de dados que representa a resposta da API AlloyDB.

### `Config`

Estrutura de dados que armazena as configurações do aplicativo.

### `LogWriter`

Estrutura de dados que implementa o escritor de log personalizado.

Constantes
------------

* `AppName`: nome do aplicativo
* `AsciiArt`: arte ASCII do aplicativo (não incluído no código fornecido)