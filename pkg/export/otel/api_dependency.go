// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Meesho fork patch P5: api_dependency_total — per-request join of a service
// instance's egress CLIENT spans to their entry (SERVER/CONSUMER) span,
// aggregated at the edge into a bounded counter:
//
//	api_dependency_total{entry_api, dst_api, server_address, parented}
//
// Ordering: client spans complete BEFORE their enclosing entry span, so
// egress spans are buffered per trace id and flushed when the entry span
// arrives (an entry span always completes after all the calls it caused).
// Traces whose entry span never arrives (background jobs, buffer eviction)
// are dropped, never guessed — the service-graph metric still covers them
// at service granularity.
//
// Bounds (cannot regress into a leak):
//   - pending traces per service instance: capped FIFO (evict oldest)
//   - buffered egress calls per trace: capped (drop excess, count overflow)
//   - distinct (entry_api, dst_api) pairs per instance: capped LRU-less set
//     (excess increments api_dependency_overflow_total instead)
package otel

import (
	"context"
	"sync"

	"go.opentelemetry.io/otel/attribute"
	trace2 "go.opentelemetry.io/otel/trace"

	instrument "go.opentelemetry.io/obi/pkg/export/otel/metric/api/metric"

	"go.opentelemetry.io/obi/pkg/appolly/app/request"
)

const (
	// APIDependencyTotal counts entry-API -> downstream-API calls.
	APIDependencyTotal = "api_dependency_total"
	// APIDependencyOverflow counts drops from the cardinality/pair cap.
	APIDependencyOverflow = "api_dependency_overflow_total"

	apiDepMaxPendingTraces = 8192 // in-flight traces buffered per service instance
	apiDepMaxCallsPerTrace = 64   // distinct egress calls buffered per trace
	apiDepMaxPairs         = 2048 // distinct (entry,dst) label pairs per instance
)

type apiDepCall struct {
	dstAPI     string
	serverAddr string
	parented   bool
}

type apiDepPending struct {
	traceID trace2.TraceID
	calls   []apiDepCall
}

// apiDepTracker is owned by one per-service-instance Metrics set; methods are
// called from the (single-goroutine) metrics record path, but a mutex keeps it
// safe if that ever changes.
type apiDepTracker struct {
	mu      sync.Mutex
	ctx     context.Context
	counter instrument.Int64Counter
	overfl  instrument.Int64Counter

	pending map[trace2.TraceID]*apiDepPending
	fifo    []trace2.TraceID // insertion order for capped eviction
	pairs   map[[2]string]struct{}
}

func newAPIDepTracker(ctx context.Context, counter, overflow instrument.Int64Counter) *apiDepTracker {
	return &apiDepTracker{
		ctx:     ctx,
		counter: counter,
		overfl:  overflow,
		pending: map[trace2.TraceID]*apiDepPending{},
		pairs:   map[[2]string]struct{}{},
	}
}

// Span feeds every span of the service instance through the tracker.
func (a *apiDepTracker) Span(span *request.Span) {
	if a == nil || !span.TraceID.IsValid() {
		return
	}
	if span.IsClientSpan() {
		a.addEgress(span)
	} else {
		a.flushEntry(span)
	}
}

func (a *apiDepTracker) addEgress(span *request.Span) {
	call := apiDepCall{
		dstAPI:     span.TraceName(),
		serverAddr: request.HTTPClientHost(span),
		parented:   span.ParentSpanID.IsValid(),
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	p := a.pending[span.TraceID]
	if p == nil {
		if len(a.fifo) >= apiDepMaxPendingTraces {
			// evict the oldest pending trace (its entry span never showed up)
			oldest := a.fifo[0]
			a.fifo = a.fifo[1:]
			delete(a.pending, oldest)
		}
		p = &apiDepPending{traceID: span.TraceID}
		a.pending[span.TraceID] = p
		a.fifo = append(a.fifo, span.TraceID)
	}
	if len(p.calls) < apiDepMaxCallsPerTrace {
		p.calls = append(p.calls, call)
	}
}

func (a *apiDepTracker) flushEntry(span *request.Span) {
	a.mu.Lock()
	p := a.pending[span.TraceID]
	if p != nil {
		delete(a.pending, span.TraceID)
		// lazy FIFO cleanup: entries for deleted traces are skipped at eviction
		// time; do a compaction sweep when the slice grows far beyond the map.
		if len(a.fifo) > 2*apiDepMaxPendingTraces {
			kept := a.fifo[:0]
			for _, id := range a.fifo {
				if _, ok := a.pending[id]; ok {
					kept = append(kept, id)
				}
			}
			a.fifo = kept
		}
	}
	a.mu.Unlock()
	if p == nil || len(p.calls) == 0 {
		return
	}
	entry := span.TraceName()
	for _, c := range p.calls {
		a.recordPair(entry, c)
	}
}

func (a *apiDepTracker) recordPair(entry string, c apiDepCall) {
	key := [2]string{entry, c.dstAPI}
	a.mu.Lock()
	_, known := a.pairs[key]
	if !known {
		if len(a.pairs) >= apiDepMaxPairs {
			a.mu.Unlock()
			a.overfl.Add(a.ctx, 1)
			return
		}
		a.pairs[key] = struct{}{}
	}
	a.mu.Unlock()
	parented := "inferred"
	if c.parented {
		parented = "w3c"
	}
	a.counter.Add(a.ctx, 1, instrument.WithAttributeSet(attribute.NewSet(
		attribute.String("entry_api", entry),
		attribute.String("dst_api", c.dstAPI),
		attribute.String("server_address", c.serverAddr),
		attribute.String("parented", parented),
	)))
}
