package signal

import "signal/internal"

type Signal = internal.Signal

type SignalDispatcher = internal.SignalDispatcher

func SignalDispatcherCreate() *SignalDispatcher {
	return internal.SignalDispatcherCreate()
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

func SignalContextSignalCreate(ctx *SignalContext, signalID string) Signal {
	return internal.SignalContextSignalCreate(ctx, signalID)
}

func SignalContextEmit(ctx *SignalContext, signal Signal) {
	internal.SignalContextEmit(ctx, signal)
}
