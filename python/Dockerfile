# Usar uma imagem base Python
FROM python:3.9-slim

# Instalar dependências necessárias
RUN apt-get update && \
    apt-get install -y curl gnupg && \
    curl https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key add - && \
    echo "deb http://packages.cloud.google.com/apt cloud-sdk main" | tee -a /etc/apt/sources.list.d/google-cloud-sdk.list && \
    apt-get update && \
    apt-get install -y google-cloud-sdk jq

# Definir diretório de trabalho
WORKDIR /app

# Copiar o script Python, arquivo de chave da conta de serviço e requirements.txt para o contêiner
COPY alloydb-autoscaler.py .
COPY tfc9924-service-9-374429.json /app/tfc9924-service-9-374429.json
COPY requirements.txt .

# Instalar dependências do Python
RUN pip install -r requirements.txt

# Definir variáveis de ambiente
ENV GOOGLE_APPLICATION_CREDENTIALS="/app/tfc9924-service-9-374429.json"

# Comando para executar o script Python
CMD ["python", "alloydb-autoscaler.py"]