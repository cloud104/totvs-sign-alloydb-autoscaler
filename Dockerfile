# Etapa de construção
FROM golang:alpine3.21 AS build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
# Compilação estática apontando para o novo caminho do main.go
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -a -installsuffix cgo -o main ./cmd/autoscaler

# Etapa final
FROM alpine:3.21

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

# Comando para executar o aplicativo binário
CMD ["/app/main"]
