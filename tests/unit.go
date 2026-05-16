package tests

import (
	"fmt"
	"foundation/location"
	"shield"
	"signal"
	"signal/rendering"
	"slices"
	"splash"
	"strings"
	"sync"
	"time"
)

type scenarioInput struct {
	signalID string

	diagnosticCategoryManifest signal.DiagnosticCategoryManifest
}

type scenarioOutput struct{}

var defaultManifest = signal.DiagnosticCategoryManifest{
	{
		Label:  "",
		Weight: 0,
	},
	{
		Label:  "ERROR",
		Weight: 10,
	},
}

func init() {
	flowOperation := shield.SHIELD_Testing_OperationCreateStateless(
		"operation_signal_to_sink",
		"Validates that a signal can flow from emission to sink properly",
		func(_ struct{}, execCtx shield.SHIELD_Testing_ExecutionContext) []shield.SHIELD_Testing_ScenarioRunResult {
			return []shield.SHIELD_Testing_ScenarioRunResult{
				runFlowScenario(execCtx),
				runIdempotencyScenario(execCtx),
				runContextScenario(execCtx),
				runContextIsolationScenario(execCtx),
				runConcurrencyScenario(execCtx),
				runPayloadAccessorScenario(execCtx),
				runTimestampScenario(execCtx),
				runContextForkConcurrencyScenario(execCtx),
				runSignalLocationScenario(execCtx),
				runBuilderScenario(execCtx),
				runBuilderKillSwitchScenario(execCtx),
			}
		},
		"SIGNAL", "core",
	)

	diagnosticCategoryOperation := shield.SHIELD_Testing_OperationCreateStateless(
		"operation_diagnostic_category",
		"Validates that the library can handle (custom) diagnostic categories properly",
		func(_ struct{}, execCtx shield.SHIELD_Testing_ExecutionContext) []shield.SHIELD_Testing_ScenarioRunResult {
			return []shield.SHIELD_Testing_ScenarioRunResult{
				runDiagnosticCategoryFlowScenario(execCtx),
				runDiagnosticCategoryUnknownManifestPanicScenario(execCtx),
				runDiagnosticDuplicateCategoryScenario(execCtx),
				runDiagnosticSinkFilterScenario(execCtx),
			}
		},
		"SIGNAL", "core", "diagnostic_categories",
	)

	rendererOperation := shield.SHIELD_Testing_OperationCreateStateless(
		"operation_signal_rendering",
		"Validates the buffered rendering subsystem",
		func(_ struct{}, execCtx shield.SHIELD_Testing_ExecutionContext) []shield.SHIELD_Testing_ScenarioRunResult {
			return []shield.SHIELD_Testing_ScenarioRunResult{
				runBufferedRendererScenario(execCtx),
			}
		},
		"SIGNAL", "render",
	)

	shield.SHIELD_Registry_OperationRegister(flowOperation)
	shield.SHIELD_Registry_OperationRegister(diagnosticCategoryOperation)
	shield.SHIELD_Registry_OperationRegister(rendererOperation)
}

func runContextForkConcurrencyScenario(execCtx shield.SHIELD_Testing_ExecutionContext) shield.SHIELD_Testing_ScenarioRunResult {
	type scenarioInput struct {
		signalID string
	}
	type scenarioOutput struct{}

	var observerSink []signal.Signal
	var observerMutex sync.Mutex

	scenario := shield.SHIELD_Testing_ScenarioCreate(
		"scenario_context_fork_concurrency",
		"Validates that forked contexts inherit state but isolate mutations safely across goroutines",
		[]shield.SHIELD_Testing_Guard[scenarioInput, scenarioOutput]{
			shield.SHIELD_Testing_GuardCreate(
				"guard_thread_safe_isolation",
				scenarioInput{signalID: "ERROR_FORK_CONCURRENT"},
				shield.SHIELD_Testing_GuardPolicyMustNotPanic[scenarioOutput](),
				shield.SHIELD_Testing_GuardPolicyPredicate(func(_ scenarioOutput) (passed bool, reason string) {
					expectedWorkers := 100

					if len(observerSink) != expectedWorkers {
						return false, fmt.Sprintf("expected %d signals, got %d", expectedWorkers, len(observerSink))
					}

					seenWorkers := make(map[string]struct{})

					for _, sig := range observerSink {
						trace := sig.SpanTrace()

						if len(trace) != 2 {
							return false, fmt.Sprintf("expected trace length 2, got %d: %v", len(trace), trace)
						}
						if trace[0] != "Root Span" {
							return false, fmt.Sprintf("expected first span to be 'Root Span', got '%s'", trace[0])
						}

						workerSpan := trace[1]
						if _, exists := seenWorkers[workerSpan]; exists {
							return false, fmt.Sprintf("cross-contamination or race detected: duplicate worker span '%s'", workerSpan)
						}
						seenWorkers[workerSpan] = struct{}{}
					}

					return true, ""
				}),
			),
		},
		func(input scenarioInput) (scenarioOutput, error) {
			dispatcher := signal.SignalDispatcherCreate(defaultManifest)

			signal.SignalDispatcherRegisterSink(dispatcher, "observer", func(sig signal.Signal) {
				observerMutex.Lock()
				defer observerMutex.Unlock()
				observerSink = append(observerSink, sig)
			})

			rootCtx := signal.SignalContextCreate(dispatcher)
			signal.SignalContextPushSpan(rootCtx, "Root Span")

			var wg sync.WaitGroup
			workers := 100
			panicChan := make(chan interface{}, 1)

			for i := 0; i < workers; i++ {
				wg.Add(1)
				go func(workerID int) {
					defer wg.Done()

					defer func() {
						if r := recover(); r != nil {
							select {
							case panicChan <- r:
							default:
							}
						}
					}()

					childCtx := signal.SignalContextClone(rootCtx)

					workerSpan := fmt.Sprintf("Worker_%d", workerID)
					signal.SignalContextPushSpan(childCtx, workerSpan)

					testSignal := signal.SignalContextSignalCreate(childCtx, input.signalID, "ERROR", nil, nil)
					signal.SignalContextEmit(childCtx, testSignal)
				}(i)
			}

			wg.Wait()

			select {
			case p := <-panicChan:
				panic(p)
			default:
			}

			rootSignal := signal.SignalContextSignalCreate(rootCtx, "ROOT_FINAL", "ERROR", nil, nil)
			rootTrace := rootSignal.SpanTrace()
			if len(rootTrace) != 1 || rootTrace[0] != "Root Span" {
				panic(fmt.Sprintf("Root context contaminated: %v", rootTrace))
			}

			return scenarioOutput{}, nil
		},
	)

	return shield.SHIELD_Testing_OperationRunScenario(
		scenario,
		execCtx,
		shield.SHIELD_Testing_ScenarioRunConfig{MaxIterations: 1},
	)
}

// --- Scenarios ---

func runFlowScenario(execCtx shield.SHIELD_Testing_ExecutionContext) shield.SHIELD_Testing_ScenarioRunResult {
	return executeTestScenario(
		execCtx,
		"scenario_basic_flow",
		"Validates a signal can go from: emit -> sink",
		"ERROR_001",
		verifyFlowData,
		executeFlowAction,
	)
}

func runIdempotencyScenario(execCtx shield.SHIELD_Testing_ExecutionContext) shield.SHIELD_Testing_ScenarioRunResult {
	return executeTestScenario(
		execCtx,
		"scenario_sink_idempotency",
		"Validates that registering the same sink key multiple times is idempotent",
		"ERROR_002",
		verifyIdempotencyData,
		executeIdempotencyAction,
	)
}

func runContextScenario(execCtx shield.SHIELD_Testing_ExecutionContext) shield.SHIELD_Testing_ScenarioRunResult {
	return executeTestScenario(
		execCtx,
		"scenario_context",
		"Validates that context correctly impacts signal span trace",
		"ERROR_003",
		verifyContextData,
		executeContextAction,
	)
}

func runContextIsolationScenario(execCtx shield.SHIELD_Testing_ExecutionContext) shield.SHIELD_Testing_ScenarioRunResult {
	return executeTestScenario(
		execCtx,
		"scenario_context_isolation",
		"Validates that context stack correctly isolates scopes after a pop",
		"ERROR_004",
		verifyContextIsolationData,
		executeContextIsolationAction,
	)
}

func runConcurrencyScenario(execCtx shield.SHIELD_Testing_ExecutionContext) shield.SHIELD_Testing_ScenarioRunResult {
	type scenarioInput struct {
		signalID string
	}
	type scenarioOutput struct{}

	scenario := shield.SHIELD_Testing_ScenarioCreate(
		"scenario_dispatcher_concurrency",
		"Validates that the dispatcher can handle concurrent registrations and emissions",
		[]shield.SHIELD_Testing_Guard[scenarioInput, scenarioOutput]{
			shield.SHIELD_Testing_GuardCreate(
				"guard_thread_safe",
				scenarioInput{signalID: "ERROR_CONCURRENT"},
				shield.SHIELD_Testing_GuardPolicyMustNotPanic[scenarioOutput](),
			),
		},
		func(input scenarioInput) (scenarioOutput, error) {
			dispatcher := signal.SignalDispatcherCreate(defaultManifest)

			var observerMutex sync.Mutex
			var emissionsCaptured int

			panicChan := make(chan interface{}, 1)

			safeSinkMethod := func(sig signal.Signal) {
				observerMutex.Lock()
				defer observerMutex.Unlock()
				emissionsCaptured++
			}

			var wg sync.WaitGroup
			workers := 100

			for i := 0; i < workers; i++ {
				wg.Add(1)
				go func(workerID int) {
					defer wg.Done()

					defer func() {
						if r := recover(); r != nil {
							select {
							case panicChan <- r:
							default:

							}
						}
					}()

					workerKey := fmt.Sprintf("worker_sink_%d", workerID)
					signal.SignalDispatcherRegisterSink(dispatcher, workerKey, safeSinkMethod)

					ctx := signal.SignalContextCreate(dispatcher)
					testSignal := signal.SignalContextSignalCreate(ctx, input.signalID, "", nil, nil)
					signal.SignalContextEmit(ctx, testSignal)
				}(i)
			}

			wg.Wait()

			select {
			case p := <-panicChan:
				panic(p)
			default:
			}

			return scenarioOutput{}, nil
		},
	)

	return shield.SHIELD_Testing_OperationRunScenario(
		scenario,
		execCtx,
		shield.SHIELD_Testing_ScenarioRunConfig{MaxIterations: 1},
	)
}

func runDiagnosticCategoryFlowScenario(execCtx shield.SHIELD_Testing_ExecutionContext) shield.SHIELD_Testing_ScenarioRunResult {
	return executeTestScenario(
		execCtx,
		"scenario_signal_diagnostic_category_flow",
		"Validates that the signal library persists diagnostic category in signal",
		"ERROR_005",
		verifyDiagCatFlowData,
		executeDiagFlowAction,
	)
}

func runDiagnosticCategoryUnknownManifestPanicScenario(execCtx shield.SHIELD_Testing_ExecutionContext) shield.SHIELD_Testing_ScenarioRunResult {
	type scenarioInput struct {
		signalID string
	}
	type scenarioOutput struct{}

	scenario := shield.SHIELD_Testing_ScenarioCreate(
		"scenario_signal_diagnostic_category_unknown",
		"Validates that the library panics when an unknown diagnostic category is presented",
		[]shield.SHIELD_Testing_Guard[scenarioInput, scenarioOutput]{
			shield.SHIELD_Testing_GuardCreate(
				"must_panic",
				scenarioInput{signalID: "ERROR_006"},
				shield.SHIELD_Testing_GuardPolicyMustPanic[scenarioOutput](),
			),
		},
		func(input scenarioInput) (scenarioOutput, error) {
			dispatcher := signal.SignalDispatcherCreate(signal.DiagnosticCategoryManifest{})
			ctx := signal.SignalContextCreate(dispatcher)
			testSignal := signal.SignalContextSignalCreate(ctx, input.signalID, "ERROR", nil, nil)
			signal.SignalContextEmit(ctx, testSignal)

			return scenarioOutput{}, nil
		},
	)

	return shield.SHIELD_Testing_OperationRunScenario(
		scenario,
		execCtx,
		shield.SHIELD_Testing_ScenarioRunConfig{MaxIterations: 1},
	)
}

func runDiagnosticDuplicateCategoryScenario(execCtx shield.SHIELD_Testing_ExecutionContext) shield.SHIELD_Testing_ScenarioRunResult {
	type scenarioInput struct {
		signalID string
	}
	type scenarioOutput struct{}

	scenario := shield.SHIELD_Testing_ScenarioCreate(
		"scenario_signal_diagnostic_category_duplicate",
		"Validates that the library panics when the manifest contains a duplicate diagnostic category label",
		[]shield.SHIELD_Testing_Guard[scenarioInput, scenarioOutput]{
			shield.SHIELD_Testing_GuardCreate(
				"must_panic",
				scenarioInput{signalID: "ERROR_007"},
				shield.SHIELD_Testing_GuardPolicyMustPanic[scenarioOutput](),
			),
		},
		func(input scenarioInput) (scenarioOutput, error) {
			_ = signal.SignalDispatcherCreate(signal.DiagnosticCategoryManifest{
				{
					Label: "ERROR",
				},
				{
					Label: "ERROR",
				},
			})

			return scenarioOutput{}, nil
		},
	)

	return shield.SHIELD_Testing_OperationRunScenario(
		scenario,
		execCtx,
		shield.SHIELD_Testing_ScenarioRunConfig{MaxIterations: 1},
	)
}

func runDiagnosticSinkFilterScenario(execCtx shield.SHIELD_Testing_ExecutionContext) shield.SHIELD_Testing_ScenarioRunResult {
	type scenarioInput struct {
		infoID  string
		errorID string
	}
	type scenarioOutput struct{}

	sink := make([]signal.Signal, 0)

	sinkMethod := func(input signal.Signal) {
		sink = append(sink, input)
	}

	scenario := shield.SHIELD_Testing_ScenarioCreate(
		"scenario_signal_diagnostic_filtered_sink",
		"Validates that the library can handle simple filtered sinks (min weight)",
		[]shield.SHIELD_Testing_Guard[scenarioInput, scenarioOutput]{
			shield.SHIELD_Testing_GuardCreate(
				"guard",
				scenarioInput{errorID: "ERROR_007", infoID: "INFO_007"},
				shield.SHIELD_Testing_GuardPolicyPredicate(func(_ scenarioOutput) (passed bool, reason string) {
					amountInSink := len(sink)

					if amountInSink == 0 {
						return false, "expected at least 1 entry in sink"
					}

					for _, entry := range sink {
						diagnosticCat := entry.DiagnosticCategory()
						if diagnosticCat != "ERROR" {
							return false, fmt.Sprintf("expected sink entry to be 'ERROR', got '%s'", diagnosticCat)
						}
					}

					return true, ""
				}),
			),
		},
		func(input scenarioInput) (scenarioOutput, error) {
			dispatcher := signal.SignalDispatcherCreate(signal.DiagnosticCategoryManifest{
				{
					Label:  "INFO",
					Weight: 0,
				},
				{
					Label:  "ERROR",
					Weight: 10,
				},
			})
			signal.SignalDispatcherRegisterFilteredSink(dispatcher, "memory", sinkMethod, 10)

			ctx := signal.SignalContextCreate(dispatcher)
			testInfoSignal := signal.SignalContextSignalCreate(ctx, input.infoID, "INFO", nil, nil)
			testErrorSignal := signal.SignalContextSignalCreate(ctx, input.errorID, "ERROR", nil, nil)

			signal.SignalContextEmit(ctx, testInfoSignal)
			signal.SignalContextEmit(ctx, testErrorSignal)

			return scenarioOutput{}, nil
		},
	)

	return shield.SHIELD_Testing_OperationRunScenario(
		scenario,
		execCtx,
		shield.SHIELD_Testing_ScenarioRunConfig{MaxIterations: 1},
	)
}

func runPayloadAccessorScenario(execCtx shield.SHIELD_Testing_ExecutionContext) shield.SHIELD_Testing_ScenarioRunResult {
	type scenarioInput struct {
		signalID string
		payload  map[string]any
	}
	type scenarioOutput struct{}

	var observerSink []signal.Signal
	var observerMutex sync.Mutex

	scenario := shield.SHIELD_Testing_ScenarioCreate(
		"scenario_signal_payload_accessors",
		"Validates the safe extraction of payload data via accessors, including type safety and missing keys",
		[]shield.SHIELD_Testing_Guard[scenarioInput, scenarioOutput]{
			shield.SHIELD_Testing_GuardCreate(
				"guard_payload_access",
				scenarioInput{
					signalID: "ERROR_PAYLOAD_ACCESS",
					payload: map[string]any{
						"retry_count": 5,
						"module_name": "auth",
					},
				},
				shield.SHIELD_Testing_GuardPolicyPredicate(func(_ scenarioOutput) (passed bool, reason string) {
					if len(observerSink) != 1 {
						return false, "expected observer sink to capture exactly 1 signal"
					}

					sig := observerSink[0]

					val, err := signal.SignalPayloadGet(&sig, "module_name")
					if err != nil {
						return false, fmt.Sprintf("expected no error for 'module_name', got: %v", err)
					}
					if val != "auth" {
						return false, fmt.Sprintf("expected 'auth', got: %v", val)
					}

					retries, err := signal.SignalPayloadGetAs[int](&sig, "retry_count")
					if err != nil {
						return false, fmt.Sprintf("expected no error for 'retry_count', got: %v", err)
					}
					if retries != 5 {
						return false, fmt.Sprintf("expected 5, got: %d", retries)
					}

					_, err = signal.SignalPayloadGet(&sig, "non_existent")
					if err == nil {
						return false, "expected error for missing key, got nil"
					}

					_, err = signal.SignalPayloadGetAs[string](&sig, "retry_count") // Requesting int as string
					if err == nil {
						return false, "expected error for type mismatch, got nil"
					}

					return true, ""
				}),
			),
		},
		func(input scenarioInput) (scenarioOutput, error) {
			dispatcher := signal.SignalDispatcherCreate(defaultManifest)

			signal.SignalDispatcherRegisterSink(dispatcher, "observer", func(sig signal.Signal) {
				observerMutex.Lock()
				defer observerMutex.Unlock()
				observerSink = append(observerSink, sig)
			})

			ctx := signal.SignalContextCreate(dispatcher)

			testSignal := signal.SignalContextSignalCreate(ctx, input.signalID, "ERROR", input.payload, nil)
			signal.SignalContextEmit(ctx, testSignal)

			return scenarioOutput{}, nil
		},
	)

	return shield.SHIELD_Testing_OperationRunScenario(
		scenario,
		execCtx,
		shield.SHIELD_Testing_ScenarioRunConfig{MaxIterations: 1},
	)
}

func runTimestampScenario(execCtx shield.SHIELD_Testing_ExecutionContext) shield.SHIELD_Testing_ScenarioRunResult {
	type scenarioInput struct {
		signalID string
	}
	type scenarioOutput struct{}

	var observerSink []signal.Signal

	scenario := shield.SHIELD_Testing_ScenarioCreate(
		"scenario_signal_timestamp",
		"Validates that a signal captures a precise creation timestamp",
		[]shield.SHIELD_Testing_Guard[scenarioInput, scenarioOutput]{
			shield.SHIELD_Testing_GuardCreate(
				"guard_timestamp",
				scenarioInput{
					signalID: "ERROR_TIMESTAMP",
				},
				shield.SHIELD_Testing_GuardPolicyPredicate(func(_ scenarioOutput) (passed bool, reason string) {
					if len(observerSink) != 1 {
						return false, "expected observer sink to capture exactly 1 signal"
					}

					sig := observerSink[0]
					stamp := sig.Timestamp()

					if stamp.IsZero() {
						return false, "expected signal timestamp to be non-zero"
					}

					delta := time.Since(stamp)
					if delta < 0 {
						return false, fmt.Sprintf("timestamp is in the future (delta: %v)", delta)
					}
					if delta > time.Second {
						return false, fmt.Sprintf("timestamp is too old, indicating it was not captured at creation (delta: %v)", delta)
					}

					return true, ""
				}),
			),
		},
		func(input scenarioInput) (scenarioOutput, error) {
			dispatcher := signal.SignalDispatcherCreate(defaultManifest)

			signal.SignalDispatcherRegisterSink(dispatcher, "observer", func(sig signal.Signal) {
				observerSink = append(observerSink, sig)
			})

			ctx := signal.SignalContextCreate(dispatcher)

			testSignal := signal.SignalContextSignalCreate(ctx, input.signalID, "ERROR", nil, nil)
			signal.SignalContextEmit(ctx, testSignal)

			return scenarioOutput{}, nil
		},
	)

	return shield.SHIELD_Testing_OperationRunScenario(
		scenario,
		execCtx,
		shield.SHIELD_Testing_ScenarioRunConfig{MaxIterations: 1},
	)
}

func runSignalLocationScenario(execCtx shield.SHIELD_Testing_ExecutionContext) shield.SHIELD_Testing_ScenarioRunResult {
	type scenarioInput struct {
		signalID string
	}
	type scenarioOutput struct{}

	var observerSink []signal.Signal
	var observerMutex sync.Mutex

	scenario := shield.SHIELD_Testing_ScenarioCreate(
		"scenario_signal_location",
		"Validates that a signal correctly captures and exposes an optional source location",
		[]shield.SHIELD_Testing_Guard[scenarioInput, scenarioOutput]{
			shield.SHIELD_Testing_GuardCreate(
				"guard_location_integrity",
				scenarioInput{signalID: "ERROR_LOCATION"},
				shield.SHIELD_Testing_GuardPolicyPredicate(func(_ scenarioOutput) (passed bool, reason string) {
					if len(observerSink) != 2 {
						return false, fmt.Sprintf("expected 2 signals, got %d", len(observerSink))
					}

					// 1. Verify Signal WITH Location
					sigWithLoc := observerSink[0]
					if !sigWithLoc.HasLocation() {
						return false, "expected first signal to report HasLocation() == true"
					}

					loc := sigWithLoc.Location()
					if loc == nil {
						return false, "expected Location() to return non-nil"
					}

					// Verify the hierarchical data survived
					if loc.Scheme() != "file" || loc.Path() != "/src/main.go" {
						return false, fmt.Sprintf("location mismatch: %s://%s", loc.Scheme(), loc.Path())
					}

					// Verify the spatial coordinates survived
					line, err := location.LocationCoordinateGetAs[int](*loc, "line")
					if err != nil || line != 42 {
						return false, "location coordinate 'line' corrupted or missing"
					}

					// 2. Verify Signal WITHOUT Location
					sigWithoutLoc := observerSink[1]
					if sigWithoutLoc.HasLocation() {
						return false, "expected second signal to report HasLocation() == false"
					}
					if sigWithoutLoc.Location() != nil {
						return false, "expected Location() to return nil for second signal"
					}

					return true, ""
				}),
			),
		},
		func(input scenarioInput) (scenarioOutput, error) {
			dispatcher := signal.SignalDispatcherCreate(defaultManifest)

			signal.SignalDispatcherRegisterSink(dispatcher, "observer", func(sig signal.Signal) {
				observerMutex.Lock()
				defer observerMutex.Unlock()
				observerSink = append(observerSink, sig)
			})

			ctx := signal.SignalContextCreate(dispatcher)

			// 1. Emit WITH location
			loc := location.LocationCreate("file", "", "/src/main.go", "", "", map[string]any{"line": 42})

			// Assuming signature: SignalContextSignalCreate(ctx, id, category, payload, locationPointer)
			testSignal1 := signal.SignalContextSignalCreate(ctx, input.signalID+"_1", "ERROR", nil, &loc)
			signal.SignalContextEmit(ctx, testSignal1)

			// 2. Emit WITHOUT location
			testSignal2 := signal.SignalContextSignalCreate(ctx, input.signalID+"_2", "ERROR", nil, nil)
			signal.SignalContextEmit(ctx, testSignal2)

			return scenarioOutput{}, nil
		},
	)

	return shield.SHIELD_Testing_OperationRunScenario(
		scenario,
		execCtx,
		shield.SHIELD_Testing_ScenarioRunConfig{MaxIterations: 1},
	)
}

func runBuilderScenario(execCtx shield.SHIELD_Testing_ExecutionContext) shield.SHIELD_Testing_ScenarioRunResult {
	type scenarioInput struct {
		baseID string
		richID string
	}
	type scenarioOutput struct{}

	var observerSink []signal.Signal
	var observerMutex sync.Mutex

	scenario := shield.SHIELD_Testing_ScenarioCreate(
		"scenario_signal_builder",
		"Validates the fluent SignalBuilder correctly accumulates state, handles lazy allocation, and emits",
		[]shield.SHIELD_Testing_Guard[scenarioInput, scenarioOutput]{
			shield.SHIELD_Testing_GuardCreate(
				"guard_builder_integrity",
				scenarioInput{
					baseID: "ERROR_BUILDER_BASE",
					richID: "ERROR_BUILDER_RICH",
				},
				shield.SHIELD_Testing_GuardPolicyPredicate(func(_ scenarioOutput) (passed bool, reason string) {
					if len(observerSink) != 2 {
						return false, fmt.Sprintf("expected 2 signals, got %d", len(observerSink))
					}

					// 1. Validate Base Case (No Payload, No Location)
					baseSig := observerSink[0]
					if baseSig.ID() != "ERROR_BUILDER_BASE" {
						return false, fmt.Sprintf("expected base ID, got '%s'", baseSig.ID())
					}
					if baseSig.HasLocation() {
						return false, "expected base signal to have no location"
					}
					_, err := signal.SignalPayloadGet(&baseSig, "any_key")
					if err == nil {
						return false, "expected error when querying empty lazy payload, got nil"
					}

					// 2. Validate Rich Case (Accumulated Payload, Attached Location)
					richSig := observerSink[1]
					if richSig.ID() != "ERROR_BUILDER_RICH" {
						return false, fmt.Sprintf("expected rich ID, got '%s'", richSig.ID())
					}

					// Validate Location
					if !richSig.HasLocation() {
						return false, "expected rich signal to have a location"
					}
					if richSig.Location().Path() != "/src/builder.go" {
						return false, "builder failed to attach correct location path"
					}

					// Validate Accumulated Payload
					strVal, err := signal.SignalPayloadGetAs[string](&richSig, "module")
					if err != nil || strVal != "auth" {
						return false, "builder failed to accumulate string payload"
					}

					intVal, err := signal.SignalPayloadGetAs[int](&richSig, "retry_count")
					if err != nil || intVal != 3 {
						return false, "builder failed to accumulate integer payload"
					}

					return true, ""
				}),
			),
		},
		func(input scenarioInput) (scenarioOutput, error) {
			dispatcher := signal.SignalDispatcherCreate(defaultManifest)

			signal.SignalDispatcherRegisterSink(dispatcher, "observer", func(sig signal.Signal) {
				observerMutex.Lock()
				defer observerMutex.Unlock()
				observerSink = append(observerSink, sig)
			})

			ctx := signal.SignalContextCreate(dispatcher)

			// 1. The Minimal Chain (Tests lazy allocation safety)
			signal.SignalContextBuild(ctx, input.baseID, "ERROR").Emit()

			// 2. The Maximal Chain (Tests state accumulation)
			loc := location.LocationCreate("file", "", "/src/builder.go", "", "", nil)
			signal.SignalContextBuild(ctx, input.richID, "ERROR").
				Payload("module", "auth").
				Payload("retry_count", 3).
				Location(&loc).
				Emit()

			return scenarioOutput{}, nil
		},
	)

	return shield.SHIELD_Testing_OperationRunScenario(
		scenario,
		execCtx,
		shield.SHIELD_Testing_ScenarioRunConfig{MaxIterations: 1},
	)
}

func runBuilderKillSwitchScenario(execCtx shield.SHIELD_Testing_ExecutionContext) shield.SHIELD_Testing_ScenarioRunResult {
	type scenarioInput struct {
		signalID string
	}
	type scenarioOutput struct{}

	scenario := shield.SHIELD_Testing_ScenarioCreate(
		"scenario_signal_builder_killswitch",
		"Validates that the SignalBuilder panics if reused after emission to prevent state contamination",
		[]shield.SHIELD_Testing_Guard[scenarioInput, scenarioOutput]{
			shield.SHIELD_Testing_GuardCreate(
				"guard_must_panic_on_reuse",
				scenarioInput{signalID: "ERROR_BUILDER_REUSE"},
				shield.SHIELD_Testing_GuardPolicyMustPanic[scenarioOutput](),
			),
		},
		func(input scenarioInput) (scenarioOutput, error) {
			dispatcher := signal.SignalDispatcherCreate(defaultManifest)
			ctx := signal.SignalContextCreate(dispatcher)

			// 1. Initialize the builder
			builder := signal.SignalContextBuild(ctx, input.signalID, "ERROR").
				Payload("key", "value")

			// 2. The first emit must succeed and drain the state
			builder.Emit()

			// 3. The second emit MUST panic (kill-switch triggered)
			builder.Emit()

			return scenarioOutput{}, nil
		},
	)

	return shield.SHIELD_Testing_OperationRunScenario(
		scenario,
		execCtx,
		shield.SHIELD_Testing_ScenarioRunConfig{MaxIterations: 1},
	)
}

func runBufferedRendererScenario(execCtx shield.SHIELD_Testing_ExecutionContext) shield.SHIELD_Testing_ScenarioRunResult {
	type scenarioInput struct{}
	type scenarioOutput struct {
		renderedOutput string
	}

	scenario := shield.SHIELD_Testing_ScenarioCreate(
		"scenario_signal_buffered_renderer",
		"Validates that the buffered renderer captures signals and formats them sequentially upon request",
		[]shield.SHIELD_Testing_Guard[scenarioInput, scenarioOutput]{
			shield.SHIELD_Testing_GuardCreate(
				"guard_render_integrity",
				scenarioInput{},
				shield.SHIELD_Testing_GuardPolicyPredicate(func(output scenarioOutput) (passed bool, reason string) {
					rendered := output.renderedOutput

					// 1. Validate that the renderer hasn't swallowed signals
					if !strings.Contains(rendered, "ERROR_PARSER_01") {
						return false, "missing first signal ID in output"
					}
					if !strings.Contains(rendered, "WARNING_LEXER_02") {
						return false, "missing second signal ID in output"
					}

					// 2. Validate that context spans were rendered
					if !strings.Contains(rendered, "AST Parsing Phase") {
						return false, "missing span trace data in output"
					}

					// 3. Validate that payload formatting was executed
					if !strings.Contains(rendered, "unexpected_token") {
						return false, "missing payload data in output"
					}

					return true, ""
				}),
			),
		},
		func(input scenarioInput) (scenarioOutput, error) {
			manifest := signal.DiagnosticCategoryManifest{
				{Label: "WARNING", Weight: 10},
				{Label: "ERROR", Weight: 20},
			}
			dispatcher := signal.SignalDispatcherCreate(manifest)

			// 1. Create a dummy palette for the test (since it uses ModeNone anyway)
			paletteBuilder := splash.SPLASH_Rendering_TerminalPaletteBuilderCreate(int(intentCount))
			palette := paletteBuilder.Build()

			// 2. Direct use of the splash dependency
			splashRenderer := splash.SPLASH_Rendering_TerminalRendererCreate(splash.SPLASH_Rendering_TerminalColorModeNone, palette)

			// 3. Inject into the adapter
			renderer := rendering.SignalRendererCreate(splashRenderer, categoryGroupingStrategy, clientLocationFormatter, nil, IntentMeta)

			signal.SignalDispatcherRegisterSink(dispatcher, "buffered_cli_renderer", rendering.SignalRendererSinkGet(renderer))

			// 4. Setup Context and emit signals
			ctx := signal.SignalContextCreate(dispatcher)
			signal.SignalContextPushSpan(ctx, "AST Parsing Phase")

			// Emit Error
			signal.SignalContextBuild(ctx, "ERROR_PARSER_01", "ERROR").
				Payload("reason", "unexpected_token").
				Emit()

			// Emit Warning
			signal.SignalContextBuild(ctx, "WARNING_LEXER_02", "WARNING").
				Emit()

			// 5. Render the buffer
			finalOutput := rendering.SignalRendererRender(renderer)

			return scenarioOutput{renderedOutput: finalOutput}, nil
		},
	)

	return shield.SHIELD_Testing_OperationRunScenario(
		scenario,
		execCtx,
		shield.SHIELD_Testing_ScenarioRunConfig{MaxIterations: 1},
	)
}

// --- Specific Scenario Behaviors ---

func verifyFlowData(sink []signal.Signal) (bool, string) {
	if len(sink) != 1 {
		return false, fmt.Sprintf("expected 1 stored signal, got %d", len(sink))
	}
	if sink[0].ID() != "ERROR_001" {
		return false, fmt.Sprintf("expected ID 'ERROR_001', got '%s'", sink[0].ID())
	}
	return true, ""
}

func executeFlowAction(dispatcher *signal.SignalDispatcher, sinkMethod func(signal.Signal), input scenarioInput) {
	signal.SignalDispatcherRegisterSink(dispatcher, "memory", sinkMethod)

	ctx := signal.SignalContextCreate(dispatcher)
	testSignal := signal.SignalContextSignalCreate(ctx, input.signalID, "", nil, nil)
	signal.SignalContextEmit(ctx, testSignal)
}

func verifyIdempotencyData(sink []signal.Signal) (bool, string) {
	if len(sink) != 1 {
		return false, fmt.Sprintf("expected exactly 1 stored signal despite duplicate registration, got %d", len(sink))
	}
	return true, ""
}

func executeIdempotencyAction(dispatcher *signal.SignalDispatcher, sinkMethod func(signal.Signal), input scenarioInput) {
	signal.SignalDispatcherRegisterSink(dispatcher, "memory", sinkMethod)
	signal.SignalDispatcherRegisterSink(dispatcher, "memory", sinkMethod)

	ctx := signal.SignalContextCreate(dispatcher)
	testSignal := signal.SignalContextSignalCreate(ctx, input.signalID, "", nil, nil)
	signal.SignalContextEmit(ctx, testSignal)
}

func verifyContextData(sink []signal.Signal) (bool, string) {
	if len(sink) != 1 {
		return false, fmt.Sprintf("expected 1 stored signal, got %d", len(sink))
	}

	spanTrace := sink[0].SpanTrace()
	if !slices.Contains(spanTrace, "Parsing Module") {
		return false, fmt.Sprintf("expected 'Parsing Module' in span trace, got %v", spanTrace)
	}

	return true, ""
}

func executeContextAction(dispatcher *signal.SignalDispatcher, sinkMethod func(signal.Signal), input scenarioInput) {
	signal.SignalDispatcherRegisterSink(dispatcher, "memory", sinkMethod)

	ctx := signal.SignalContextCreate(dispatcher)

	signal.SignalContextPushSpan(ctx, "Parsing Module")

	testSignal := signal.SignalContextSignalCreate(ctx, input.signalID, "", nil, nil)
	signal.SignalContextEmit(ctx, testSignal)

	signal.SignalContextPopSpan(ctx)
}

func verifyContextIsolationData(sink []signal.Signal) (bool, string) {
	if len(sink) != 1 {
		return false, fmt.Sprintf("expected 1 stored signal, got %d", len(sink))
	}

	spanTrace := sink[0].SpanTrace()

	if !slices.Contains(spanTrace, "Phase A") {
		return false, fmt.Sprintf("expected 'Phase A' in span trace, got %v", spanTrace)
	}

	if slices.Contains(spanTrace, "Phase B") {
		return false, fmt.Sprintf("did not expect 'Phase B' in span trace after pop, got %v", spanTrace)
	}

	return true, ""
}

func executeContextIsolationAction(dispatcher *signal.SignalDispatcher, sinkMethod func(signal.Signal), input scenarioInput) {
	signal.SignalDispatcherRegisterSink(dispatcher, "memory", sinkMethod)

	ctx := signal.SignalContextCreate(dispatcher)

	signal.SignalContextPushSpan(ctx, "Phase A")
	signal.SignalContextPushSpan(ctx, "Phase B")

	signal.SignalContextPopSpan(ctx)

	testSignal := signal.SignalContextSignalCreate(ctx, input.signalID, "", nil, nil)
	signal.SignalContextEmit(ctx, testSignal)
}

func verifyDiagCatFlowData(sink []signal.Signal) (bool, string) {
	if len(sink) != 1 {
		return false, fmt.Sprintf("expected 1 stored signal, got %d", len(sink))
	}

	diagnosticCategory := sink[0].DiagnosticCategory()

	if diagnosticCategory != "ERROR" {
		return false, fmt.Sprintf("expected diagnostic category to be 'ERROR', got '%s'", diagnosticCategory)
	}

	return true, ""
}

func executeDiagFlowAction(dispatcher *signal.SignalDispatcher, sinkMethod func(signal.Signal), input scenarioInput) {
	signal.SignalDispatcherRegisterSink(dispatcher, "memory", sinkMethod)

	ctx := signal.SignalContextCreate(dispatcher)
	testSignal := signal.SignalContextSignalCreate(ctx, input.signalID, "ERROR", nil, nil)
	signal.SignalContextEmit(ctx, testSignal)
}

// --- Framework Abstraction ---

func executeTestScenario(
	execCtx shield.SHIELD_Testing_ExecutionContext,
	name string,
	description string,
	signalID string,
	verifyLogic func(sink []signal.Signal) (bool, string),
	actionLogic func(dispatcher *signal.SignalDispatcher, sinkMethod func(signal.Signal), input scenarioInput),
) shield.SHIELD_Testing_ScenarioRunResult {

	sink := []signal.Signal{}
	sinkMethod := func(input signal.Signal) {
		sink = append(sink, input)
	}

	scenario := shield.SHIELD_Testing_ScenarioCreate(
		name,
		description,
		[]shield.SHIELD_Testing_Guard[scenarioInput, scenarioOutput]{
			createGuard(signalID, &sink, verifyLogic),
		},
		func(input scenarioInput) (scenarioOutput, error) {
			dispatcher := signal.SignalDispatcherCreate(input.diagnosticCategoryManifest)
			actionLogic(dispatcher, sinkMethod, input)
			return scenarioOutput{}, nil
		},
	)

	return shield.SHIELD_Testing_OperationRunScenario(
		scenario,
		execCtx,
		shield.SHIELD_Testing_ScenarioRunConfig{MaxIterations: 1},
	)
}

func createGuard(
	signalID string,
	sink *[]signal.Signal,
	verifyLogic func(sink []signal.Signal) (bool, string),
) shield.SHIELD_Testing_Guard[scenarioInput, scenarioOutput] {

	return shield.SHIELD_Testing_GuardCreate(
		"guard",
		scenarioInput{signalID: signalID, diagnosticCategoryManifest: defaultManifest},
		shield.SHIELD_Testing_GuardPolicyPredicate(
			func(_ scenarioOutput) (bool, string) {
				return verifyLogic(*sink)
			},
		),
	)
}
