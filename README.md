# AlloyDB Autoscaler
======================

## Descrição
------------

O AlloyDB Autoscaler é um aplicativo que escala automaticamente o número de réplicas de leitura de um cluster AlloyDB com base no uso de CPU e memória.

## Variáveis de Ambiente
-------------------------

As seguintes variáveis de ambiente são essenciais para o funcionamento do aplicativo:

* `GOOGLE_APPLICATION_CREDENTIALS`: caminho para o arquivo de credenciais do Google Cloud
* `GCP_PROJECT`: ID do projeto do Google Cloud
* `CLUSTER_NAME`: nome do cluster AlloyDB
* `INSTANCE_NAME`: nome da instância de leitura do AlloyDB
* `REGION`: região onde o cluster AlloyDB está localizado
* `CHECK_INTERVAL`: intervalo de tempo (em segundos) entre as checagens do uso de CPU e memória
* `CPU_THRESHOLD`: limiar de uso de CPU (em porcentagem, apenas o número, exemplo: 50)
* `MEMORY_THRESHOLD`: valor de memória livre desejável (em GB, apenas o número, exemplo: 50)
* `MIN_REPLICAS`: número mínimo de réplicas permitidas
* `MAX_REPLICAS`: número máximo de réplicas permitidas (máximo 20, de acordo com as limitações da GCP)

## Funcionamento
----------------

O aplicativo funciona da seguinte maneira:

1. Lê as variáveis de ambiente e configura o cliente do Google Cloud
2. Verifica o uso de CPU e memória do cluster AlloyDB a cada intervalo de tempo especificado
3. Se o uso de CPU ultrapassar o limiar especificado ou a memória livre for inferior ao valor desejável, escala o número de réplicas do cluster
4. Se o uso de CPU e memória estiver abaixo do limiar especificado e a memória livre for superior ao valor desejável, reduz o número de réplicas do cluster

## Requisitos
--------------

* Go 1.17 ou superior
* Google Cloud SDK instalado e configurado
* AlloyDB cluster criado e configurado

## Instalação
--------------

1. Clone o repositório do aplicativo
2. Execute o comando `go build` para compilar o aplicativo
3. Execute o comando `go run` para executar o aplicativo

## Exemplos de Uso
--------------------

* `CHECK_INTERVAL=300 CPU_THRESHOLD=50 MEMORY_THRESHOLD=50 MIN_REPLICAS=1 MAX_REPLICAS=10 go run main.go`
* `CHECK_INTERVAL=600 CPU_THRESHOLD=80 MEMORY_THRESHOLD=100 MIN_REPLICAS=2 MAX_REPLICAS=20 go run main.go`

### Uso com Docker

Para usar o aplicativo com Docker, siga os passos abaixo:

1. Preencha o arquivo `.env` com as variáveis de ambiente necessárias
2. Execute o comando `docker build -t autoscaler .` para construir a imagem do Docker
3. Execute o comando `docker run -v ./.env:/app/.env -v ./caminho_do_arquivo/key.json:/caminho_do_arquivo/key.json autoscaler` para executar o aplicativo
4. Execute o comando `docker logs -f -t autoscaler` para visualizar os logs do aplicativo

IMPORTANTE: A regra para desescalar só vale se o nodepoolcount for maior que 1, se existir apenas 1 node que é o minimo, o programa só vai considerar a possibilidade de escala.