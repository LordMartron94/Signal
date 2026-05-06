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
	registeredSinks map[string]struct{}
	sinks           []SignalSink
}

func SignalDispatcherCreate() *SignalDispatcher {
	return &SignalDispatcher{
		registeredSinks: make(map[string]struct{}),
		sinks:           make([]SignalSink, 0),
	}
}

type SignalSink = func(input Signal)

func SignalDispatcherRegisterSink(dispatcher *SignalDispatcher, key string, sink SignalSink) {
	if _, exist := dispatcher.registeredSinks[key]; !exist {
		dispatcher.sinks = append(dispatcher.sinks, sink)
		dispatcher.registeredSinks[key] = struct{}{}
	}
}

func SignalDispatcherEmit(dispatcher *SignalDispatcher, signal Signal) {
	for _, sink := range dispatcher.sinks {
		sink(signal)
	}
}
