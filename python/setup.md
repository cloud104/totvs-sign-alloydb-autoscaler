# Cria a imagem docker localmente:
  - docker build -t alloydb-autoscaler .

# Executa o programa local para testes:
  - Primeiro vá no .env local e defina os valores como desejar nas variáveis de ambiente.
  - adicione a credencial .json  da service account da gcp na raiz do projeto
  - docker run --env-file .env -v ./tfc9924-service-9-374429.json:/tfc9924-service-9-374429.json alloydb-autoscaler && docker logs -f -t alloydb-autoscaler


# Cria a secret com a SA da Google no kubernetes local
  - kubectl create secret generic gcp-key-secret --from-file=key.json=service-account-file.json