package internal

// ----------------------------------------------------------- SIGNAL

type Signal struct {
	id        string
	spanTrace []string
}

func (s *Signal) ID() string {
	return s.id
}

func (s *Signal) SpanTrace() []string {
	return s.spanTrace
}

func SignalCreate(id string) Signal {
	return Signal{
		id:        id,
		spanTrace: make([]string, 0),
	}
}

// ----------------------------------------------------------- SIGNAL DISPATCHER

type SignalDispatcher struct {
	registeredSinks map[string]struct{}
	sinks           []SignalSink

	spanStack []string
}

func SignalDispatcherCreate() *SignalDispatcher {
	return &SignalDispatcher{
		registeredSinks: make(map[string]struct{}),
		sinks:           make([]SignalSink, 0),
		spanStack:       make([]string, 0),
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
	copiedTrace := make([]string, len(dispatcher.spanStack))
	copy(copiedTrace, dispatcher.spanStack)
	signal.spanTrace = copiedTrace

	for _, sink := range dispatcher.sinks {
		sink(signal)
	}
}

func SignalDispatcherPushSpan(dispatcher *SignalDispatcher, span string) {
	dispatcher.spanStack = append(dispatcher.spanStack, span)
}

func SignalDispatcherPopSpan(dispatcher *SignalDispatcher) {
	amountOfSpans := len(dispatcher.spanStack)
	if amountOfSpans > 0 {
		dispatcher.spanStack = dispatcher.spanStack[0 : amountOfSpans-1]
	}
}
