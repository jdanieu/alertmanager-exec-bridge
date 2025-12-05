package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/jdanieu/alertmanager-exec-bridge/internal/config"
	"github.com/jdanieu/alertmanager-exec-bridge/internal/executor"
)

const maxBodyBytes = 1 << 20 // 1 MiB

func firstLine(stderr string, err error) string {
	if stderr != "" {
		if idx := strings.Index(stderr, "\n"); idx != -1 {
			return stderr[:idx]
		}
		return stderr
	}
	if err != nil {
		return err.Error()
	}
	return "unknown error"
}

func Run(cfg config.Config, logger *slog.Logger) error {
	mux := http.NewServeMux()

	// Healthcheck
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// Handler de Alertmanager
	mux.HandleFunc("/alert", func(w http.ResponseWriter, r *http.Request) {

		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		// Validación de token en cabecera X-Token (si está configurado)
		if cfg.Token != "" {
			token := r.Header.Get("X-Token")
			if token == "" || token != cfg.Token {
				logger.Warn("unauthorized request: invalid or missing token",
					"remote_addr", r.RemoteAddr,
				)
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
		}

		// Limitar tamaño del body
		r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
		defer r.Body.Close()

		body, err := io.ReadAll(r.Body)
		if err != nil {
			logger.Error("failed to read request body",
				"error", err,
				"remote_addr", r.RemoteAddr,
			)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		var payload AlertmanagerPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			logger.Error("failed to parse alertmanager payload",
				"error", err,
				"remote_addr", r.RemoteAddr,
			)
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}

		if err := payload.Validate(); err != nil {
			logger.Error("invalid alertmanager payload",
				"error", err,
				"remote_addr", r.RemoteAddr,
			)
			http.Error(w, "invalid payload", http.StatusBadRequest)
			return
		}

		// Renderizar comando y argumentos
		cmdPath, cmdArgs, err := executor.RenderCommand(cfg.Command, cfg.Args, &payload)
		if err != nil {
			logger.Error("failed to render command from template",
				"error", err,
				"remote_addr", r.RemoteAddr,
			)
			http.Error(w, "template error", http.StatusInternalServerError)
			return
		}

		logger.Info("alert mapped to command",
			"status", payload.Status,
			"receiver", payload.Receiver,
			"group_key", payload.GroupKey,
			"alerts_count", len(payload.Alerts),
			"primary_alertname", payload.PrimaryAlertName(),
			"command", cmdPath,
			"args", cmdArgs,
			"remote_addr", r.RemoteAddr,
		)

		// Ejecutar el comando de forma síncrona: esta request espera al resultado,
		// pero el servidor sigue atendiendo otras en paralelo.
		res, err := executor.Run(cmdPath, cmdArgs, cfg.Timeout)

		// Caso error / fallo / timeout
		if err != nil || res.ExitCode != 0 || res.TimedOut {
			logger.Error("command execution failed",
				"command", res.Command,
				"args", res.Args,
				"exit_code", res.ExitCode,
				"timed_out", res.TimedOut,
				"duration_ms", res.Duration.Milliseconds(),
				"stdout", res.Stdout,
				"stderr", res.Stderr,
				"error", err,
				"primary_alertname", payload.PrimaryAlertName(),
				"alerts_count", len(payload.Alerts),
			)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			resp := map[string]any{
				"status":    "error",
				"exit_code": res.ExitCode,
				"timed_out": res.TimedOut,
				"message":   firstLine(res.Stderr, err),
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		// Caso éxito
		logger.Info("command execution succeeded",
			"command", res.Command,
			"args", res.Args,
			"exit_code", res.ExitCode,
			"timed_out", res.TimedOut,
			"duration_ms", res.Duration.Milliseconds(),
			"stdout", res.Stdout,
			"stderr", res.Stderr,
			"primary_alertname", payload.PrimaryAlertName(),
			"alerts_count", len(payload.Alerts),
		)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		resp := map[string]any{
			"status":    "ok",
			"exit_code": res.ExitCode,
			"timed_out": res.TimedOut,
		}
		_ = json.NewEncoder(w).Encode(resp)

	})

	srv := &http.Server{
		Addr:              cfg.Listen,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	logger.Info("http server starting", "addr", cfg.Listen)

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("http server error: %w", err)
	}

	logger.Info("http server stopped cleanly")
	return nil
}
