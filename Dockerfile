# Etapa 1: build
FROM golang:1.25 AS builder

WORKDIR /app

# Copiamos go.mod y go.sum primero para aprovechar la cache
COPY go.mod go.sum ./
RUN go mod download

# Copiar el resto del código
COPY . .

# Compilar binario estático para Linux
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /alert-exec ./cmd/alert-exec

# Etapa 2: runtime
FROM alpine:3.20

# Añadir usuario no root
RUN adduser -D -H -s /sbin/nologin alertexec

USER alertexec

WORKDIR /app

# Copiar binario
COPY --from=builder /alert-exec /app/alert-exec

# Puerto de escucha (informativo)
EXPOSE 9095

ENTRYPOINT ["/app/alert-exec"]
