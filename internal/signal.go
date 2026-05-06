package internal

// ----------------------------------------------------------- SIGNAL

type Signal struct {
	id string
}

func (s *Signal) ID() string {
	return s.id
}

func SignalCreate(id string) Signal {
	return Signal{
		id: id,
	}
}

// ----------------------------------------------------------- SIGNAL DISPATCHER

type SignalDispatcher struct {
	sinks []SignalSink
}

func SignalDispatcherCreate() *SignalDispatcher {
	return &SignalDispatcher{
		sinks: make([]SignalSink, 0),
	}
}

type SignalSink = func(input Signal)

func SignalDispatcherRegisterSink(dispatcher *SignalDispatcher, sink SignalSink) {
	dispatcher.sinks = append(dispatcher.sinks, sink)
}

func SignalDispatcherEmit(dispatcher *SignalDispatcher, signal Signal) {
	for _, sink := range dispatcher.sinks {
		sink(signal)
	}
}
