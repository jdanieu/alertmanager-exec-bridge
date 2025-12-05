package server

import (
	"fmt"
	"time"
)

// AlertmanagerPayload representa el JSON que manda Alertmanager al webhook.
type AlertmanagerPayload struct {
	Version           string            `json:"version"`
	GroupKey          string            `json:"groupKey"`
	TruncatedAlerts   int               `json:"truncatedAlerts"`
	Status            string            `json:"status"`
	Receiver          string            `json:"receiver"`
	GroupLabels       map[string]string `json:"groupLabels"`
	CommonLabels      map[string]string `json:"commonLabels"`
	CommonAnnotations map[string]string `json:"commonAnnotations"`
	ExternalURL       string            `json:"externalURL"`
	Alerts            []Alert           `json:"alerts"`
}

type Alert struct {
	Status       string            `json:"status"`
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
	StartsAt     time.Time         `json:"startsAt"`
	EndsAt       time.Time         `json:"endsAt"`
	GeneratorURL string            `json:"generatorURL"`
	Fingerprint  string            `json:"fingerprint"`
}

// Validate hace una validación mínima del payload.
func (p *AlertmanagerPayload) Validate() error {
	if p.Status == "" {
		return fmt.Errorf("status is required")
	}
	if len(p.Alerts) == 0 {
		return fmt.Errorf("at least one alert is required")
	}
	return nil
}

// Helper para sacar un alertname "representativo"
func (p *AlertmanagerPayload) PrimaryAlertName() string {
	if len(p.Alerts) == 0 {
		return ""
	}
	if name, ok := p.Alerts[0].Labels["alertname"]; ok {
		return name
	}
	return ""
}
