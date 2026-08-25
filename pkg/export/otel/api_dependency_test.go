// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otel

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	trace2 "go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/obi/pkg/appolly/app/request"
	"go.opentelemetry.io/otel/metric/embedded"

	instrument "go.opentelemetry.io/obi/pkg/export/otel/metric/api/metric"
)

type fakeCounter struct {
	embedded.Int64Counter
	adds []map[string]string
}

func (f *fakeCounter) Add(_ context.Context, _ int64, opts ...instrument.AddOption) {
	labels := map[string]string{}
	cfg := instrument.NewAddConfig(opts)
	set := cfg.Attributes()
	for _, kv := range set.ToSlice() {
		labels[string(kv.Key)] = kv.Value.AsString()
	}
	f.adds = append(f.adds, labels)
}

func (f *fakeCounter) Remove(_ context.Context, _ ...instrument.RemoveOption) {}

func tidN(i int) trace2.TraceID {
	return trace2.TraceID{byte(i), byte(i >> 8), byte(i >> 16), 1}
}
func tid(b byte) trace2.TraceID   { return trace2.TraceID{b, 1} }
func sid(b byte) trace2.SpanID    { return trace2.SpanID{b, 1} }
func attrsOf(m map[string]string) attribute.Set {
	kvs := make([]attribute.KeyValue, 0, len(m))
	for k, v := range m {
		kvs = append(kvs, attribute.String(k, v))
	}
	return attribute.NewSet(kvs...)
}

func clientSpan(t trace2.TraceID, name, host string, parented bool) *request.Span {
	s := &request.Span{
		Type:    request.EventTypeHTTPClient,
		Method:  "GET",
		Path:    name,
		Route:   name, // routes decoration runs before the metrics stage
		TraceID: t,
		SpanID:  sid(9),
		Host:    host,
	}
	if parented {
		s.ParentSpanID = sid(7)
	}
	return s
}

func serverSpan(t trace2.TraceID, name string) *request.Span {
	return &request.Span{
		Type:    request.EventTypeHTTP,
		Method:  "POST",
		Path:    name,
		Route:   name,
		TraceID: t,
		SpanID:  sid(7),
	}
}

func newTestTracker() (*apiDepTracker, *fakeCounter, *fakeCounter) {
	c, o := &fakeCounter{}, &fakeCounter{}
	return newAPIDepTracker(context.Background(), c, o), c, o
}

func TestAPIDep_JoinsEgressToEntry(t *testing.T) {
	tr, c, _ := newTestTracker()

	// two egress calls complete first, then the entry span
	tr.Span(clientSpan(tid(1), "/inventory", "svc-b", true))
	tr.Span(clientSpan(tid(1), "/stock", "svc-c", false))
	assert.Empty(t, c.adds, "nothing recorded before the entry span arrives")

	tr.Span(serverSpan(tid(1), "/checkout"))
	require.Len(t, c.adds, 2)
	assert.Equal(t, "POST /checkout", c.adds[0]["entry_api"])
	assert.Equal(t, "GET /inventory", c.adds[0]["dst_api"])
	assert.Equal(t, "svc-b", c.adds[0]["server_address"])
	assert.Equal(t, "w3c", c.adds[0]["parented"])
	assert.Equal(t, "GET /stock", c.adds[1]["dst_api"])
	assert.Equal(t, "inferred", c.adds[1]["parented"])
}

func TestAPIDep_TraceIsolation(t *testing.T) {
	tr, c, _ := newTestTracker()
	tr.Span(clientSpan(tid(1), "/a", "x", true))
	tr.Span(clientSpan(tid(2), "/b", "y", true))
	tr.Span(serverSpan(tid(2), "/entry2"))
	require.Len(t, c.adds, 1)
	assert.Equal(t, "GET /b", c.adds[0]["dst_api"])
	// trace 1 still pending; its entry arrives later
	tr.Span(serverSpan(tid(1), "/entry1"))
	require.Len(t, c.adds, 2)
	assert.Equal(t, "GET /a", c.adds[1]["dst_api"])
}

func TestAPIDep_EntryWithoutEgressIsQuiet(t *testing.T) {
	tr, c, _ := newTestTracker()
	tr.Span(serverSpan(tid(3), "/lonely"))
	assert.Empty(t, c.adds)
}

func TestAPIDep_PendingTraceCapEvictsOldest(t *testing.T) {
	tr, c, _ := newTestTracker()
	for i := 0; i < apiDepMaxPendingTraces+1; i++ {
		tr.Span(clientSpan(tidN(i), "/x", "h", true))
	}
	// oldest trace evicted: its entry span now joins nothing
	tr.Span(serverSpan(tidN(0), "/entry"))
	assert.Empty(t, c.adds)
	// newest trace still pending and flushable
	tr.Span(serverSpan(tidN(apiDepMaxPendingTraces), "/entry"))
	assert.Len(t, c.adds, 1)
}

func TestAPIDep_PairCapOverflows(t *testing.T) {
	tr, c, o := newTestTracker()
	for i := 0; i <= apiDepMaxPairs; i++ {
		id := tidN(i + 1000000)
		tr.Span(clientSpan(id, string(rune('a'+i%26))+string(rune('a'+(i/26)%26))+string(rune('a'+(i/676)%26)), "h", true))
		tr.Span(serverSpan(id, "/entry"))
	}
	assert.Len(t, c.adds, apiDepMaxPairs, "recording stops at the pair cap")
	assert.Len(t, o.adds, 1, "overflow counted")
}

func TestAPIDep_InvalidTraceIgnored(t *testing.T) {
	tr, c, _ := newTestTracker()
	s := clientSpan(trace2.TraceID{}, "/x", "h", true)
	tr.Span(s)
	tr.Span(serverSpan(trace2.TraceID{}, "/entry"))
	assert.Empty(t, c.adds)
}

// keep the linter honest about the helper
var _ = attrsOf
