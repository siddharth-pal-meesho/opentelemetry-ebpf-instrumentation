// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

#pragma once

#include <bpfcore/vmlinux.h>
#include <bpfcore/bpf_helpers.h>

#include <common/map_sizing.h>
#include <common/pin_internal.h>
#include <common/tp_info.h>

// Correlates in-flight server requests with client requests of the same
// process by trace id. Complements the thread-keyed server_traces map for
// runtimes where the server and client side of one request run on different
// threads (e.g. JVM async frameworks): there, in-process instrumentation
// carries the trace id across the async boundary and writes it into the
// outgoing traceparent header, where the client-side probes read it back.
typedef struct trace_id_key {
    u32 pid; // host process id
    unsigned char trace_id[TRACE_ID_SIZE_BYTES];
} trace_id_key_t;

struct {
    __uint(type, BPF_MAP_TYPE_LRU_HASH);
    __type(key, trace_id_key_t);
    __type(value, tp_info_pid_t); // the server request's traceparent info
    __uint(max_entries, MAX_CONCURRENT_SHARED_REQUESTS);
    __uint(pinning, OBI_PIN_INTERNAL);
} trace_id_server_map SEC(".maps");

// Live server-request count per process, remembering the latest request:
// when a client request's trace id matches no live server request (the
// process is a trace entry point whose in-process instrumentation minted its
// own downstream trace id), a single live server request is still an
// unambiguous parent. With more than one, no parent is guessed.
typedef struct pid_server_state {
    u32 live;
    u8 _pad[4];
    tp_info_pid_t tp;
} pid_server_state_t;

struct {
    __uint(type, BPF_MAP_TYPE_LRU_HASH);
    __type(key, u32); // host process id
    __type(value, pid_server_state_t);
    __uint(max_entries, MAX_CONCURRENT_SHARED_REQUESTS);
    __uint(pinning, OBI_PIN_INTERNAL);
} pid_server_state_map SEC(".maps");
