// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package procs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnvStrParsing(t *testing.T) {
	strs := []string{
		"OTEL_SERVICE_NAME=\"=  =\"",
		"nothing",
		"=wrong",
		"TMPDIR=somethingelse",
		"CLASSPATH=",
		"TMPDIR= else",
		"OTEL_EXPORTER_OTLP_ENDPOINT==  =",
		"OTEL_RESOURCE_ATTRIBUTES=a=b,c=d,e=  fg",
		"",
	}

	res := envStrsToMap(strs)
	assert.Equal(t, map[string]string{"TMPDIR": "else", "OTEL_SERVICE_NAME": "\"=  =\"", "OTEL_EXPORTER_OTLP_ENDPOINT": "=  =", "OTEL_RESOURCE_ATTRIBUTES": "a=b,c=d,e=  fg"}, res)
}

func TestEnvStrParsingDropsUnlistedVars(t *testing.T) {
	strs := []string{
		"OTEL_SERVICE_NAME=my-service",
		"JAVA_OPTS=-Xmx4g -Xms4g -XX:+UseG1GC",
		"KUBERNETES_SERVICE_HOST=10.0.0.1",
		"PATH=/usr/bin:/bin",
		"SOME_APP_SECRET=abcd",
	}

	res := envStrsToMap(strs)
	assert.Equal(t, map[string]string{"OTEL_SERVICE_NAME": "my-service"}, res)
}
