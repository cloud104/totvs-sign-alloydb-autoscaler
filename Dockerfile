# Etapa de construção
FROM golang:1.22-alpine AS build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -a -installsuffix cgo -o main .     

# Etapa final
FROM scratch

WORKDIR /app

COPY --from=build /app/main /app/main
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

ENV TZ=America/Sao_Paulo
ENV LC_ALL=pt_BR.UTF-8
ENV LANG=pt_BR.UTF-8
ENV LANGUAGE=pt_BR.UTF-8

CMD ["/app/main"]
