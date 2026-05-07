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

func signalDispatcherEmit(dispatcher *SignalDispatcher, signal Signal) {
	for _, sink := range dispatcher.sinks {
		sink(signal)
	}
}

// ----------------------------------------------------------- CONTEXT

type SignalContext struct {
	dispatcher *SignalDispatcher
	spanStack  []string
}

func SignalContextCreate(dispatcher *SignalDispatcher) *SignalContext {
	return &SignalContext{
		dispatcher: dispatcher,
		spanStack:  make([]string, 0),
	}
}

func SignalContextPushSpan(ctx *SignalContext, span string) {
	ctx.spanStack = append(ctx.spanStack, span)
}

func SignalContextPopSpan(ctx *SignalContext) {
	amountOfSpans := len(ctx.spanStack)
	if amountOfSpans > 0 {
		ctx.spanStack = ctx.spanStack[0 : amountOfSpans-1]
	}
}

func SignalContextSignalCreate(ctx *SignalContext, signalID string) Signal {
	traceCopy := make([]string, len(ctx.spanStack))
	copy(traceCopy, ctx.spanStack)

	return Signal{
		id:        signalID,
		spanTrace: traceCopy,
	}
}

func SignalContextEmit(ctx *SignalContext, signal Signal) {
	signalDispatcherEmit(ctx.dispatcher, signal)
}
