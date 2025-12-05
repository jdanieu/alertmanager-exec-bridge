app := "alert-exec"
cmd := "./cmd/alert-exec"

build:
    go build -o bin/{{app}} {{cmd}}

run:
    go run {{cmd}} --config configs/default.yaml

docker-build:
    docker build -t jdanieu/alert-exec:local .

docker-run:
    docker run --rm -p 9095:9095 jdanieu/alert-exec:local --listen :9095

tidy:
    go mod tidy

fmt:
    go fmt ./...

test:
    go test ./...
