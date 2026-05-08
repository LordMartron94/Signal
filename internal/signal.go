package internal

import (
	"errors"
	"fmt"
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

// ----------------------------------------------------------- SIGNAL DISPATCHER

type SignalDispatcher struct {
	mutex                     sync.RWMutex
	registeredSinks           map[string]struct{}
	sinks                     []SignalSink
	manifest                  DiagnosticCategoryManifest
	validDiagnosticCategories map[DiagnosticCategory]struct{}
}

func SignalDispatcherCreate(manifest DiagnosticCategoryManifest) *SignalDispatcher {
	validDiagnosticCategories := make(map[DiagnosticCategory]struct{})
	var duplicateError error

	for _, setting := range manifest {
		if _, seen := validDiagnosticCategories[setting.Label]; seen {
			duplicateError = errors.Join(duplicateError, fmt.Errorf("duplicate diagnostic category '%s'", setting.Label))
		}
		validDiagnosticCategories[setting.Label] = struct{}{}
	}

	if duplicateError != nil {
		panic(duplicateError)
	}

	return &SignalDispatcher{
		registeredSinks:           make(map[string]struct{}),
		sinks:                     make([]SignalSink, 0),
		mutex:                     sync.RWMutex{},
		manifest:                  manifest,
		validDiagnosticCategories: validDiagnosticCategories,
	}
}

type SignalSink = func(input Signal)

func SignalDispatcherRegisterSink(dispatcher *SignalDispatcher, key string, sink SignalSink) {
	dispatcher.mutex.Lock()
	defer dispatcher.mutex.Unlock()

	if _, exist := dispatcher.registeredSinks[key]; !exist {
		dispatcher.sinks = append(dispatcher.sinks, sink)
		dispatcher.registeredSinks[key] = struct{}{}
	}
}

func signalDispatcherEmit(dispatcher *SignalDispatcher, signal Signal) {
	dispatcher.mutex.RLock()
	defer dispatcher.mutex.RUnlock()

	for _, sink := range dispatcher.sinks {
		sink(signal)
	}
}

func signalDispatcherDiagnosticCategoryIsValid(dispatcher *SignalDispatcher, category DiagnosticCategory) bool {
	if _, valid := dispatcher.validDiagnosticCategories[category]; valid {
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

func SignalContextSignalCreate(ctx *SignalContext, signalID string, diagnosticCategory DiagnosticCategory) Signal {
	if !signalDispatcherDiagnosticCategoryIsValid(ctx.dispatcher, diagnosticCategory) {
		panic(fmt.Errorf("unknown diagnostic category '%s', did you forget to declare it in the manifest?", diagnosticCategory))
	}

	traceCopy := make([]string, len(ctx.spanStack))
	copy(traceCopy, ctx.spanStack)

	return Signal{
		id:                 signalID,
		spanTrace:          traceCopy,
		diagnosticCategory: diagnosticCategory,
	}
}

func SignalContextEmit(ctx *SignalContext, signal Signal) {
	signalDispatcherEmit(ctx.dispatcher, signal)
}
