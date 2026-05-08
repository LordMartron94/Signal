package signal

import "signal/internal"

type DiagnosticCategory = internal.DiagnosticCategory

type DiagnosticCategoryManifest = internal.DiagnosticCategoryManifest

type Signal = internal.Signal

func SignalPayloadGet(signal *Signal, key string) (value any, error error) {
	return internal.SignalPayloadGet(signal, key)
}

func SignalPayloadGetAs[TValue any](signal *Signal, key string) (value TValue, error error) {
	return internal.SignalPayloadGetAs[TValue](signal, key)
}

type SignalDispatcher = internal.SignalDispatcher

func SignalDispatcherCreate(manifest DiagnosticCategoryManifest) *SignalDispatcher {
	return internal.SignalDispatcherCreate(manifest)
}

type SignalSink = internal.SignalSink

func SignalDispatcherRegisterSink(dispatcher *SignalDispatcher, key string, sink SignalSink) {
	internal.SignalDispatcherRegisterSink(dispatcher, key, sink, 0)
}

func SignalDispatcherRegisterFilteredSink(dispatcher *SignalDispatcher, key string, sink SignalSink, minWeight int) {
	internal.SignalDispatcherRegisterSink(dispatcher, key, sink, minWeight)
}

type SignalContext = internal.SignalContext

func SignalContextCreate(dispatcher *SignalDispatcher) *SignalContext {
	return internal.SignalContextCreate(dispatcher)
}

func SignalContextPushSpan(ctx *SignalContext, span string) {
	internal.SignalContextPushSpan(ctx, span)
}

func SignalContextPopSpan(ctx *SignalContext) {
	internal.SignalContextPopSpan(ctx)
}

func SignalContextSignalCreate(ctx *SignalContext, signalID string, diagnosticCategory DiagnosticCategory, payload map[string]any) Signal {
	return internal.SignalContextSignalCreate(ctx, signalID, diagnosticCategory, payload)
}

func SignalContextEmit(ctx *SignalContext, signal Signal) {
	internal.SignalContextEmit(ctx, signal)
}
