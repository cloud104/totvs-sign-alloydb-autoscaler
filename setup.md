# Cria a imagem docker localmente:
  - docker build -t autoscaler .

# Executa o programa local para testes:
  - Primeiro vá no .env local e defina os valores como desejar nas variáveis de ambiente.
  
  - adicione a credencial .json  da service account da gcp na raiz do projeto
  
  - docker run -v ./.env:/app/.env -v ./tfc9924-service-9-374429.json:/app/tfc9924-service-9-374429.json autoscaler && docker logs -f -t autoscaler


# Cria a secret com a SA da Google no kubernetes local
  - kubectl create secret generic gcp-key-secret --from-file=key.json=service-account-file.json

vou te passar um codigo em Golang, preciso de modificações. Com base no valor que o usuario informou de quanto em quanto tempo é para fazer uma checagem de CPU e memoria para escalar ou desescalar um pool do Alloydb, é necessario que exista mais uma variavel informando um tempo, por exemplo, foi efetuada verificação, e foi estipulado que se em 600 segundos (10 minutos) a CPU estiver acima de 90% é para escalar, o mesmo vale para desescalar, se em 10 minutos estiver abaixo de 90% é para desescalar obedecendo o minimo de pool estipulado pelo usuario. nao importa se foram feitas 20 verificacoes, a soma do tempo de todas elas precisa ser o tempo evaluation que foi estipulado pelo usuario, e só vai escalar para cima ou para baixo se por exemplo todas as verificações do periodo foram para escalar ou desescalar.

código: