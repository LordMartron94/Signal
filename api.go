package signal

import "signal/internal"

type DiagnosticCategory = internal.DiagnosticCategory

type DiagnosticCategoryManifest = internal.DiagnosticCategoryManifest

type Signal = internal.Signal

type SignalDispatcher = internal.SignalDispatcher

func SignalDispatcherCreate(manifest DiagnosticCategoryManifest) *SignalDispatcher {
	return internal.SignalDispatcherCreate(manifest)
}

type SignalSink = internal.SignalSink

func SignalDispatcherRegisterSink(dispatcher *SignalDispatcher, key string, sink SignalSink) {
	internal.SignalDispatcherRegisterSink(dispatcher, key, sink)
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

func SignalContextSignalCreate(ctx *SignalContext, signalID string, diagnosticCategory DiagnosticCategory) Signal {
	return internal.SignalContextSignalCreate(ctx, signalID, diagnosticCategory)
}

func SignalContextEmit(ctx *SignalContext, signal Signal) {
	internal.SignalContextEmit(ctx, signal)
}
