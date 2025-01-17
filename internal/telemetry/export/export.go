// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package export holds the definition of the telemetry Exporter interface,
// along with some simple implementations.
// Larger more complex exporters are in sub packages of their own.
package export

import (
	"context"
	"os"
	"time"

	"golang.org/x/tools/internal/telemetry"
)

type Exporter interface {
	StartSpan(context.Context, *telemetry.Span)
	FinishSpan(context.Context, *telemetry.Span)

	// Log is a function that handles logging events.
	// Observers may use information in the context to decide what to do with a
	// given log event. They should return true if they choose to handle the
	Log(context.Context, telemetry.Event)

	Metric(context.Context, telemetry.MetricData)
}

var exporter = LogWriter(os.Stderr, true)

func SetExporter(setter func(Exporter) Exporter) {
	Do(func() {
		exporter = setter(exporter)
	})
}

func AddExporters(e ...Exporter) {
	Do(func() {
		exporter = Multi(append([]Exporter{exporter}, e...)...)
	})
}

func StartSpan(ctx context.Context, span *telemetry.Span, at time.Time) {
	Do(func() {
		span.Start = at
		exporter.StartSpan(ctx, span)
	})
}

func FinishSpan(ctx context.Context, span *telemetry.Span, at time.Time) {
	Do(func() {
		span.Finish = at
		exporter.FinishSpan(ctx, span)
	})
}

func Tag(ctx context.Context, at time.Time, tags telemetry.TagList) {
	Do(func() {
		// If context has a span we need to add the tags to it
		span := telemetry.GetSpan(ctx)
		if span == nil {
			return
		}
		if span.Start.IsZero() {
			// span still being created, tag it directly
			span.Tags = append(span.Tags, tags...)
			return
		}
		// span in progress, add an event to the span
		span.Events = append(span.Events, telemetry.Event{
			At:   at,
			Tags: tags,
		})
	})
}

func Log(ctx context.Context, event telemetry.Event) {
	Do(func() {
		// If context has a span we need to add the event to it
		span := telemetry.GetSpan(ctx)
		if span != nil {
			span.Events = append(span.Events, event)
		}
		// and now also hand the event of to the current observer
		exporter.Log(ctx, event)
	})
}

func Metric(ctx context.Context, data telemetry.MetricData) {
	Do(func() {
		exporter.Metric(ctx, data)
	})
}
