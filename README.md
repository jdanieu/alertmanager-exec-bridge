# alertmanager-exec-bridge

A lightweight, stateless HTTP bridge that receives alerts from Alertmanager, transforms them through Go templates, and executes user-defined commands with proper timeout, logging, and HTTP success/error semantics.

---

## Motivation

Alertmanager ships with excellent integrations (email, Slack, PagerDuty, webhooks), but many real-world environments require executing **arbitrary commands** when an alert fires. For example:

* Triggering on-prem automations
* Sending messages through a custom API
* Running scripts or operational tooling
* Integrating with systems that do not expose webhook endpoints

This project fills that gap by providing a **small, safe, configurable, Go-based HTTP service** that accepts Alertmanager webhooks and turns them into controllable command executions.

The philosophy is:

* Define *exactly* which command to run.
* Use Go templates to render arguments from the incoming alert.
* Enforce execution timeout.
* Emit structured JSON logs.
* Return success or failure to Alertmanager via HTTP.

---

## Intended Use Case

### Deploy as an Alertmanager Sidecar

`alertmanager-exec-bridge` is designed to run **in the same Pod** as Alertmanager. Alertmanager calls it via:

```
url: http://127.0.0.1:9095/alert
```

Because it is stateless and configuration-driven, the container can be restarted at any time without losing context.

### Why a Sidecar?

* No external network dependencies
* No need to expose a public endpoint
* Low latency between Alertmanager and the bridge
* Isolated responsibility: Alertmanager routes alerts, bridge executes logic

### Stateless by Design

The service does not persist:

* alert state
* execution history
* configuration

All configuration is provided via command-line flags, environment variables, or a mounted config file.

--

## Reliability & Failure Modes

Deploying this bridge as an Alertmanager sidecar introduces an additional hop in the alert delivery pipeline, but it does **not** have to introduce a new silent single point of failure.

The failure model should be:

- If the sidecar container crashes or fails its liveness/readiness checks, the **entire Alertmanager Pod becomes NotReady** and is removed from the Service endpoints.
- Operationally, this instance is treated exactly as “Alertmanager down” and must be covered by your existing fallback / secondary alerting channel (e.g. email, PagerDuty, another Alertmanager route).

In other words, the bridge becomes part of the **health contract** of the Alertmanager Pod: if the bridge is not healthy, this instance is not considered a valid backend.

There are still two important failure modes you must monitor explicitly:

- The bridge process is running, but returns `5xx` to `/alert` (delivery failures to the external system).
- Alertmanager is healthy, but the external receiver (e.g. Evolution API) is unavailable.

Both conditions should be covered by additional alerts (e.g. based on Alertmanager metrics about failed notifications, or logs from the bridge), and by having at least one **independent notification channel** for “Alertmanager / alert pipeline is broken”.


---

## Quick Start

### Run Locally

```
go run ./cmd/alert-exec --config configs/default.yaml
```

### Send a Test Alert

```
curl -X POST http://localhost:9095/alert \
  -H 'Content-Type: application/json' \
  -d '{
    "version": "4",
    "status": "firing",
    "receiver": "test",
    "groupKey": "{}:{}",
    "alerts": [
      {
        "status": "firing",
        "labels": {"alertname": "TestAlert"},
        "annotations": {"summary": "Demo"},
        "startsAt": "2025-01-01T00:00:00Z"
      }
    ]
  }'
```

### Example Log Output

```
{"level":"INFO","msg":"alert mapped to command", ...}
{"level":"INFO","msg":"command execution succeeded", ...}
```

---

## Configuration

Configuration values come from three sources (priority descending):

1. Command-line arguments
2. Environment variables (`ALERT_EXEC_*`)
3. YAML config file

### Full Configuration Reference

#### `listen` (string)

Address to bind the HTTP server.

```
listen: ":9095"
```

#### `token` (string, optional)

Shared secret expected in the `X-Token` header.
Empty value disables authentication.

```
token: "my-secret"
```

#### `command` (string)

Path to the executable or script. Can be templated.

```
command: "/bin/echo"
```

#### `args` (list of strings)

Arguments to pass to the executable. Each entry supports Go templates.

```
args:
  - '{{ .Status }}'
  - '{{ .PrimaryAlertName }}'
```

#### `timeout` (duration)

Execution timeout for the command.

```
timeout: "5s"
```

#### `log_level` (string)

Valid values: `debug`, `info`, `warn`, `error`.

```
log_level: "info"
```

---

## HTTP Endpoints

### `GET /healthz`

Health probe. Always returns `200 OK`.

### `POST /alert`

Consumes the Alertmanager webhook format. Returns:

* `200 OK` if the command executed successfully
* `503 Service Unavailable` if execution failed or timed out

Body includes structured JSON describing the outcome.

---


## Container Image

The official container image is published automatically to GitHub Container Registry (GHCR).

### Image Location

```
ghcr.io/${{ github.repository }}:latest
```

Replace `${{ github.repository }}` with your actual repository path, e.g.:

```
ghcr.io/jdanieu/alertmanager-exec-bridge:latest
```

### Available Tags

All published tags are visible here:

```
https://ghcr.io/jdanieu/alertmanager-exec-bridge
```

(or directly in the *Packages* section of the GitHub repository)

### Running the Image Locally

Basic usage:

```
docker run --rm -p 9095:9095 ghcr.io/jdanieu/alertmanager-exec-bridge:latest \
  --listen :9095 \
  --command /bin/echo
```

Run with a configuration file mounted:

```
docker run --rm -p 9095:9095 \
  -v $(pwd)/configs/default.yaml:/config.yaml \
  ghcr.io/jdanieu/alertmanager-exec-bridge:latest \
  --config /config.yaml
```

You can now POST alerts exactly as in local development.

---

## Execution Behavior

* Validates incoming JSON using typed structs
* Renders `command` + `args` using Go templates
* Executes the command synchronously
* Enforces timeout via `executor.Run`
* Captures stdout, stderr, exit code, and duration
* Logs everything in JSON via `slog`
* Returns success/failure to Alertmanager with appropriate HTTP codes

---

## Example: Deploying as Sidecar in Kubernetes

```yaml
containers:
  - name: alert-exec
    image: jdanieu/alert-exec:latest
    args:
      - "--config=/config/config.yaml"
    ports:
      - containerPort: 9095
    volumeMounts:
      - name: alert-exec-config
        mountPath: /config

volumes:
  - name: alert-exec-config
    configMap:
      name: alert-exec-config
```

Alertmanager configuration:

```yaml
webhook_configs:
  - url: "http://127.0.0.1:9095/alert"
    http_config:
      headers:
        X-Token: "my-secret"
```


---

## Contributing

Issues and pull requests are welcome. Keep the code minimal, clear, and production-safe.
