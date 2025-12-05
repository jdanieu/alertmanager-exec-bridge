package main

import (
	"flag"
	"os"
	"time"

	"github.com/jdanieu/alertmanager-exec-bridge/internal/config"
	"github.com/jdanieu/alertmanager-exec-bridge/internal/logging"
	"github.com/jdanieu/alertmanager-exec-bridge/internal/server"
)

func main() {
	// Flags de línea de comandos
	configFile := flag.String("config", "", "Path to config file (YAML)")
	listenFlag := flag.String("listen", "", "Address to listen on, e.g. :9095")
	commandFlag := flag.String("command", "", "Command to execute")
	tokenFlag := flag.String("token", "", "Shared secret token expected in requests")
	timeoutFlag := flag.String("timeout", "", "Command timeout, e.g. 5s, 1m")
	logLevelFlag := flag.String("log-level", "", "Log level: debug, info, warn, error")

	flag.Parse()

	// Bootstrap logger (JSON, nivel info por defecto)
	bootstrapLogger := logging.New("info")

	// Cargar config (defaults + file + env)
	cfg, err := config.Load(*configFile)
	if err != nil {
		bootstrapLogger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// Overlay: flags por encima de todo lo demás
	if *listenFlag != "" {
		cfg.Listen = *listenFlag
	}
	if *commandFlag != "" {
		cfg.Command = *commandFlag
	}
	if *tokenFlag != "" {
		cfg.Token = *tokenFlag
	}
	if *timeoutFlag != "" {
		dur, err := time.ParseDuration(*timeoutFlag)
		if err != nil {
			bootstrapLogger.Error("invalid timeout flag", "value", *timeoutFlag, "error", err)
			os.Exit(1)
		}
		cfg.Timeout = dur
		cfg.TimeoutRaw = *timeoutFlag
	}
	if *logLevelFlag != "" {
		cfg.LogLevel = *logLevelFlag
	}

	// Logger final con el nivel configurado
	logger := logging.New(cfg.LogLevel)

	logger.Info("starting alert-exec",
		"listen", cfg.Listen,
		"command", cfg.Command,
		"timeout", cfg.Timeout.String(),
		"has_token", cfg.Token != "",
	)

	// Arrancar servidor HTTP (bloqueante)
	if err := server.Run(cfg, logger); err != nil {
		logger.Error("server exited with error", "error", err)
		os.Exit(1)
	}
}
