package executor

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
)

// RenderCommand renderiza el comando y los argumentos a partir de plantillas
// y los datos proporcionados (normalmente el payload de Alertmanager).
func RenderCommand(commandTemplate string, argTemplates []string, data any) (string, []string, error) {
	cmd, err := renderMaybeTemplate("command", commandTemplate, data)
	if err != nil {
		return "", nil, fmt.Errorf("rendering command template: %w", err)
	}

	args := make([]string, 0, len(argTemplates))
	for _, raw := range argTemplates {
		arg, err := renderMaybeTemplate("arg", raw, data)
		if err != nil {
			return "", nil, fmt.Errorf("rendering arg template %q: %w", raw, err)
		}
		args = append(args, arg)
	}

	return cmd, args, nil
}

func renderMaybeTemplate(name, raw string, data any) (string, error) {
	if !containsTemplate(raw) {
		return raw, nil
	}

	tmpl, err := template.New(name).Parse(raw)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func containsTemplate(s string) bool {
	return strings.Contains(s, "{{")
}
