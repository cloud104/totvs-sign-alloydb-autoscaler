# Etapa de construção
FROM golang:alpine3.20 AS build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
# Compilação estática
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -a -installsuffix cgo -o main .

# Etapa para baixar o gcloud CLI
# FROM alpine:3.15.11 AS gcloud

# RUN apk add --no-cache curl python3
# RUN curl -O https://dl.google.com/dl/cloudsdk/channels/rapid/downloads/google-cloud-cli-484.0.0-linux-x86_64.tar.gz && \
#     tar -xzf google-cloud-cli-484.0.0-linux-x86_64.tar.gz && \
#     ./google-cloud-sdk/install.sh --quiet --command-completion false --path-update false --usage-reporting false && \
#     rm -rf google-cloud-cli-484.0.0-linux-x86_64.tar.gz \
#            /google-cloud-sdk/.install \
#            /google-cloud-sdk/platform/gsutil         

# Etapa final
FROM alpine:3.15.11

WORKDIR /app

# Instalar dependências mínimas necessárias e tzdata
RUN apk add --no-cache ca-certificates tzdata

# Configurar o fuso horário e locale
ENV TZ=America/Sao_Paulo
ENV LC_ALL=pt_BR.UTF-8
ENV LANG=pt_BR.UTF-8
ENV LANGUAGE=pt_BR.UTF-8

RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

# Copiar o binário compilado
COPY --from=build /app/main /app/main

# # Copiar o Google Cloud SDK
# COPY --from=gcloud /google-cloud-sdk /google-cloud-sdk

# # Definir variáveis de ambiente necessárias
# ENV PATH /app:/google-cloud-sdk/bin:$PATH
# ENV CLOUDSDK_PYTHON /usr/bin/python3

# Comando para executar o aplicativo binário
CMD ["/app/main"]