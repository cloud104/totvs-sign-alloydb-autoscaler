# Cria a imagem docker localmente:
  docker build -t alloydb-autoscaler:latest .

# Executa o programa local para testes:
  - Primeiro vá no .env local e defina os valores como desejar nas variáveis de ambiente.
  
  - adicione a credencial .json  da service account da gcp na raiz do projeto
  
  - docker run -v ./.env:/app/.env -v ./key.json:/app/key.json alloydb-autoscaler && docker logs -f -t alloydb-autoscaler


# Cria a secret com a SA da Google no kubernetes local
  - kubectl create secret generic gcp-key-secret --from-file=key.json=service-account-file.json