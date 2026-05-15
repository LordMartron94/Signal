package signal

import (
	"foundation/location"
	"signal/internal"
)

/*
DiagnosticCategory identifies a logical diagnostics class (for example TRACE, INFO, WARNING, ERROR).

[Context]
Diagnostic categories are configured through a manifest at dispatcher creation time and are used as routing metadata for sinks.
*/
type DiagnosticCategory = internal.DiagnosticCategory

/*
DiagnosticCategoryManifest defines the ordered category configuration consumed by the dispatcher.

[Context]
Each entry provides a label and a weight. Weights are used by filtered sinks to decide whether a signal should be delivered.
*/
type DiagnosticCategoryManifest = internal.DiagnosticCategoryManifest

/*
Signal is the immutable diagnostics event payload propagated through the dispatcher pipeline.

[Context]
Signals are created from a `SignalContext` to capture span trace, timestamp, category, optional source location, and payload in one value object.
*/
type Signal = internal.Signal

/*
SignalPayloadGet returns the payload entry for `key`.

[Returns]
Returns the stored value when the key exists.

[Errors]
Returns an error when the key is missing.

[Side Effects]
Pure function. No side effects.
*/
func SignalPayloadGet(signal *Signal, key string) (value any, error error) {
	return internal.SignalPayloadGet(signal, key)
}

/*
SignalPayloadGetAs returns the payload entry for `key` casted to `TValue`.

[Returns]
Returns the typed value when the key exists and the dynamic type matches `TValue`.

[Errors]
Returns an error when the key is missing or when the stored value has a different type than `TValue`.

[Side Effects]
Pure function. No side effects.
*/
func SignalPayloadGetAs[TValue any](signal *Signal, key string) (value TValue, error error) {
	return internal.SignalPayloadGetAs[TValue](signal, key)
}

/*
SignalDispatcher is the category-aware fan-out router for signals.

[Context]
A dispatcher stores registered sinks and routes emitted signals to them based on category weight filters.
*/
type SignalDispatcher = internal.SignalDispatcher

/*
SignalDispatcherCreate builds a dispatcher from a diagnostic category manifest.

[Panics]
Panics when the manifest declares duplicate category labels.

[Side Effects]
Allocates internal routing state and category lookup tables.
*/
func SignalDispatcherCreate(manifest DiagnosticCategoryManifest) *SignalDispatcher {
	return internal.SignalDispatcherCreate(manifest)
}

/*
SignalSink is the callback signature for signal consumers.

[Context]
Sinks are executed synchronously during emission and receive a snapshot `Signal` value.
*/
type SignalSink = internal.SignalSink

/*
SignalDispatcherRegisterSink registers a sink without a minimum category weight filter.

[Context]
This is equivalent to `SignalDispatcherRegisterFilteredSink` with `minWeight = 0`.
Registration is idempotent per key.
*/
func SignalDispatcherRegisterSink(dispatcher *SignalDispatcher, key string, sink SignalSink) {
	internal.SignalDispatcherRegisterSink(dispatcher, key, sink, 0)
}

/*
SignalDispatcherRegisterFilteredSink registers a sink with a minimum category weight filter.

[Context]
Only signals whose category weight is greater than or equal to `minWeight` are delivered to this sink.
Registration is idempotent per key.
*/
func SignalDispatcherRegisterFilteredSink(dispatcher *SignalDispatcher, key string, sink SignalSink, minWeight int) {
	internal.SignalDispatcherRegisterSink(dispatcher, key, sink, minWeight)
}

/*
SignalContext is a mutable emission scope holding span state and dispatcher linkage.

[Context]
Contexts let producers push/pop spans and create consistent signals without manually re-building trace metadata each time.
*/
type SignalContext = internal.SignalContext

/*
SignalContextCreate creates a fresh context bound to `dispatcher`.

[Returns]
Returns an empty-span context ready for signal creation and emission.
*/
func SignalContextCreate(dispatcher *SignalDispatcher) *SignalContext {
	return internal.SignalContextCreate(dispatcher)
}

/*
SignalContextClone creates a new context that shares the dispatcher and copies the current span stack.

[Context]
Useful when branching execution while preserving the current diagnostic provenance.
*/
func SignalContextClone(ctx *SignalContext) *SignalContext {
	return internal.SignalContextClone(ctx)
}

/*
SignalContextPushSpan appends a span segment to the context stack.

[Side Effects]
Mutates `ctx` by extending its span stack.
*/
func SignalContextPushSpan(ctx *SignalContext, span string) {
	internal.SignalContextPushSpan(ctx, span)
}

/*
SignalContextPopSpan removes the most recent span segment from the context stack.

[Side Effects]
Mutates `ctx` by shortening its span stack when non-empty. No-op when empty.
*/
func SignalContextPopSpan(ctx *SignalContext) {
	internal.SignalContextPopSpan(ctx)
}

/*
SignalContextSignalCreate creates a signal snapshot from the current context state.

[Parameters]
`location` is optional. Pass nil when the signal has no source location (for example global or process-level diagnostics).

[Returns]
Returns a new signal containing copied span trace, copied payload map, and the provided location pointer (which may be nil).

[Panics]
Panics when `diagnosticCategory` is not declared in the dispatcher's manifest.
*/
func SignalContextSignalCreate(ctx *SignalContext, signalID string, diagnosticCategory DiagnosticCategory, payload map[string]any, location *location.Location) Signal {
	return internal.SignalContextSignalCreate(ctx, signalID, diagnosticCategory, payload, location)
}

/*
SignalContextEmit dispatches `signal` to all matching sinks registered on the context dispatcher.

[Side Effects]
Invokes sink callbacks synchronously as part of emission.
*/
func SignalContextEmit(ctx *SignalContext, signal Signal) {
	internal.SignalContextEmit(ctx, signal)
}

/*
SignalBuilder provides a fluent interface for constructing and emitting signals.

[Context]
This builder mitigates the ergonomic friction of the underlying factory function.
It lazily allocates the payload map to prevent unnecessary heap pressure when no payload is attached.
*/
type SignalBuilder struct {
	ctx      *SignalContext
	id       string
	category DiagnosticCategory
	payload  map[string]any
	location *location.Location
}

/*
SignalContextBuild initiates a fluent builder chain for signal creation.

[Returns]
Returns a pointer to a new SignalBuilder state container.
*/
func SignalContextBuild(ctx *SignalContext, signalID string, diagnosticCategory DiagnosticCategory) *SignalBuilder {
	return &SignalBuilder{
		ctx:      ctx,
		id:       signalID,
		category: diagnosticCategory,
		payload:  nil, // Lazy initialization
		location: nil,
	}
}

/*
Payload attaches a key-value pair to the signal's diagnostic data.

[Side Effects]
Mutates the builder's internal payload map, allocating it on the first call.
Returns the builder instance for chaining.
*/
func (b *SignalBuilder) Payload(key string, value any) *SignalBuilder {
	if b.payload == nil {
		b.payload = make(map[string]any)
	}
	b.payload[key] = value
	return b
}

/*
Location attaches a source location to the signal.

[Side Effects]
Mutates the builder's internal location pointer. Returns the builder instance for chaining.
*/
func (b *SignalBuilder) Location(loc *location.Location) *SignalBuilder {
	b.location = loc
	return b
}

/*
Build delegates to the core factory method to construct the immutable signal snapshot.

[Returns]
Returns the final Signal value object.

[Panics]
Panics if the diagnostic category is not declared in the dispatcher's manifest.
*/
func (b *SignalBuilder) Build() Signal {
	return SignalContextSignalCreate(b.ctx, b.id, b.category, b.payload, b.location)
}

/*
Emit builds the signal and immediately dispatches it to all matching sinks.

[Side Effects]
Invokes sink callbacks synchronously as part of emission.
*/
func (b *SignalBuilder) Emit() {
	signal := b.Build()
	SignalContextEmit(b.ctx, signal)
}
