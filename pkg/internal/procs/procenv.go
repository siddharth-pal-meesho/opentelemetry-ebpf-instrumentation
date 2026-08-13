// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package procs // import "go.opentelemetry.io/obi/pkg/internal/procs"

import (
	"strings"

	"github.com/prometheus/procfs"

	"go.opentelemetry.io/obi/pkg/appolly/app"
)

// capturedEnvVars is the allowlist of process environment variables kept
// after discovery. The captured map is retained (per process, referenced from
// every span) for the process lifetime, so capturing the full environment
// grows the live heap with #processes × environment size and OOMs nodes
// running many large-environment processes (e.g. JVMs). Only the variables
// actually consumed elsewhere in the codebase are kept.
var capturedEnvVars = map[string]struct{}{
	"OTEL_SERVICE_NAME":                   {}, // discover/exec service naming
	"OTEL_RESOURCE_ATTRIBUTES":            {}, // discover/exec + otelcfg resource attrs
	"OTEL_EXPORTER_OTLP_PROTOCOL":         {}, // request/span OTel-export detection
	"OTEL_EXPORTER_OTLP_TRACES_PROTOCOL":  {},
	"OTEL_EXPORTER_OTLP_METRICS_PROTOCOL": {},
	"OTEL_EXPORTER_OTLP_ENDPOINT":         {},
	"OTEL_EXPORTER_OTLP_TRACES_ENDPOINT":  {},
	"OTEL_EXPORTER_OTLP_METRICS_ENDPOINT": {},
	"TMPDIR":                              {}, // java agent injector temp dir
	"CLASSPATH":                           {}, // java route harvester
}

func envStrsToMap(varsStr []string) map[string]string {
	vars := make(map[string]string, len(capturedEnvVars))

	for _, s := range varsStr {
		if key, val, ok := parseEnvVar(s); ok {
			if _, keep := capturedEnvVars[key]; keep {
				vars[key] = val
			}
		}
	}

	return vars
}

func validEnvCount(varsStr []string) int {
	count := 0
	for _, s := range varsStr {
		if _, _, ok := parseEnvVar(s); ok {
			count++
		}
	}
	return count
}

func parseEnvVar(s string) (string, string, bool) {
	key, val, found := strings.Cut(s, "=")
	if !found {
		return "", "", false
	}

	key = strings.TrimSpace(key)
	val = strings.TrimSpace(val)
	return key, val, key != "" && val != ""
}

func EnvVars(pid app.PID) (map[string]string, error) {
	proc, err := procfs.NewProc(int(pid))
	if err != nil {
		return nil, err
	}

	varsStr, err := proc.Environ()
	if err != nil {
		return nil, err
	}

	m := envStrsToMap(varsStr)

	return m, nil
}
