package signal

import "signal/internal"

type Signal = internal.Signal

func SignalCreate(id string) Signal {
	return internal.SignalCreate(id)
}

type SignalDispatcher = internal.SignalDispatcher

func SignalDispatcherCreate() *SignalDispatcher {
	return internal.SignalDispatcherCreate()
}

type SignalSink = internal.SignalSink

func SignalDispatcherRegisterSink(dispatcher *SignalDispatcher, sink SignalSink) {
	internal.SignalDispatcherRegisterSink(dispatcher, sink)
}

func SignalDispatcherEmit(dispatcher *SignalDispatcher, signal Signal) {
	internal.SignalDispatcherEmit(dispatcher, signal)
}
