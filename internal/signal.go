package internal

import (
	"errors"
	"fmt"
	"maps"
	"sync"
)

// ----------------------------------------------------------- DIAGNOSTIC CATEGORY

type DiagnosticCategory string

type DiagnosticCategorySetting struct {
	Label  DiagnosticCategory
	Weight int
}

type DiagnosticCategoryManifest []DiagnosticCategorySetting

// ----------------------------------------------------------- SIGNAL

type Signal struct {
	id                 string
	spanTrace          []string
	diagnosticCategory DiagnosticCategory
	payload            map[string]any
}

func (s *Signal) ID() string {
	return s.id
}

func (s *Signal) SpanTrace() []string {
	return s.spanTrace
}

func (s *Signal) DiagnosticCategory() DiagnosticCategory {
	return s.diagnosticCategory
}

func SignalPayloadGet(signal *Signal, key string) (value any, error error) {
	if value, exist := signal.payload[key]; exist {
		return value, nil
	}

	return nil, fmt.Errorf("key '%s' not present in payload", key)
}

func SignalPayloadGetAs[TValue any](signal *Signal, key string) (TValue, error) {
	var zero TValue

	raw, err := SignalPayloadGet(signal, key)
	if err != nil {
		return zero, err
	}

	casted, ok := raw.(TValue)
	if !ok {
		return zero, fmt.Errorf("payload key '%s': expected type %T, got %T", key, zero, raw)
	}

	return casted, nil
}

// ----------------------------------------------------------- SIGNAL DISPATCHER

type SignalDispatcher struct {
	mutex                sync.RWMutex
	registeredSinks      map[string]struct{}
	sinks                []signalSinkConfiguration
	diagnosticCategories map[DiagnosticCategory]DiagnosticCategorySetting
}

func SignalDispatcherCreate(manifest DiagnosticCategoryManifest) *SignalDispatcher {
	diagnosticCategories := make(map[DiagnosticCategory]DiagnosticCategorySetting)
	var duplicateError error

	for _, setting := range manifest {
		if _, seen := diagnosticCategories[setting.Label]; seen {
			duplicateError = errors.Join(duplicateError, fmt.Errorf("duplicate diagnostic category '%s'", setting.Label))
		}
		diagnosticCategories[setting.Label] = setting
	}

	if duplicateError != nil {
		panic(duplicateError)
	}

	return &SignalDispatcher{
		registeredSinks:      make(map[string]struct{}),
		sinks:                make([]signalSinkConfiguration, 0),
		mutex:                sync.RWMutex{},
		diagnosticCategories: diagnosticCategories,
	}
}

type SignalSink = func(input Signal)

type signalSinkConfiguration struct {
	sink      SignalSink
	minWeight int
}

func SignalDispatcherRegisterSink(dispatcher *SignalDispatcher, key string, sink SignalSink, minWeight int) {
	dispatcher.mutex.Lock()
	defer dispatcher.mutex.Unlock()

	if _, exist := dispatcher.registeredSinks[key]; !exist {
		dispatcher.sinks = append(dispatcher.sinks, signalSinkConfiguration{
			sink:      sink,
			minWeight: minWeight,
		})
		dispatcher.registeredSinks[key] = struct{}{}
	}
}

func signalDispatcherEmit(dispatcher *SignalDispatcher, signal Signal) {
	dispatcher.mutex.RLock()
	defer dispatcher.mutex.RUnlock()

	// TODO - think about a better way to handle this
	// without having signal carry weight data as that is absurd

	diagnosticCategory := signal.DiagnosticCategory()
	setting := dispatcher.diagnosticCategories[diagnosticCategory]
	weight := setting.Weight

	for _, sink := range dispatcher.sinks {
		if sink.minWeight > weight {
			continue
		}

		sink.sink(signal)
	}
}

func signalDispatcherDiagnosticCategoryIsValid(dispatcher *SignalDispatcher, category DiagnosticCategory) bool {
	if _, valid := dispatcher.diagnosticCategories[category]; valid {
		return true
	}

	return false
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

func SignalContextSignalCreate(ctx *SignalContext, signalID string, diagnosticCategory DiagnosticCategory, payload map[string]any) Signal {
	if !signalDispatcherDiagnosticCategoryIsValid(ctx.dispatcher, diagnosticCategory) {
		panic(fmt.Errorf("unknown diagnostic category '%s', did you forget to declare it in the manifest?", diagnosticCategory))
	}

	traceCopy := make([]string, len(ctx.spanStack))
	copy(traceCopy, ctx.spanStack)

	payloadCopy := maps.Clone(payload)

	return Signal{
		id:                 signalID,
		spanTrace:          traceCopy,
		diagnosticCategory: diagnosticCategory,
		payload:            payloadCopy,
	}
}

func SignalContextEmit(ctx *SignalContext, signal Signal) {
	signalDispatcherEmit(ctx.dispatcher, signal)
}
