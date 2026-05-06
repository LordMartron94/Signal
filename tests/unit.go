package tests

import (
	"fmt"
	"shield"
	"signal"
	"slices"
)

type scenarioInput struct {
	testSignal signal.Signal
}

type scenarioOutput struct{}

func init() {
	operation := shield.SHIELD_Testing_OperationCreateStateless(
		"operation_signal_to_sink",
		"Validates that a signal can flow from emission to sink properly",
		func(_ struct{}, execCtx shield.SHIELD_Testing_ExecutionContext) []shield.SHIELD_Testing_ScenarioRunResult {
			return []shield.SHIELD_Testing_ScenarioRunResult{
				runFlowScenario(execCtx),
				runIdempotencyScenario(execCtx),
				runContextScenario(execCtx),
			}
		},
		"SIGNAL", "core",
	)

	shield.SHIELD_Registry_OperationRegister(operation)
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
	signal.SignalDispatcherEmit(dispatcher, input.testSignal)
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
	signal.SignalDispatcherEmit(dispatcher, input.testSignal)
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
	signal.SignalDispatcherPushSpan(dispatcher, "Parsing Module")
	signal.SignalDispatcherEmit(dispatcher, input.testSignal)
	signal.SignalDispatcherPopSpan(dispatcher)
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
			dispatcher := signal.SignalDispatcherCreate()
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
		scenarioInput{testSignal: signal.SignalCreate(signalID)},
		shield.SHIELD_Testing_GuardPolicyPredicate(
			func(_ scenarioOutput) (bool, string) {
				return verifyLogic(*sink)
			},
		),
	)
}
