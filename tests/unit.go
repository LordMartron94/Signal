package tests

import (
	"fmt"
	"shield"
	"signal"
)

func init() {
	operation := shield.SHIELD_Testing_OperationCreateStateless(
		"operation_signal_to_sink",
		"Validates that a signal can flow from emission to sink properly",
		func(_ struct{}, execCtx shield.SHIELD_Testing_ExecutionContext) []shield.SHIELD_Testing_ScenarioRunResult {
			return []shield.SHIELD_Testing_ScenarioRunResult{
				runFlowScenario(execCtx),
			}
		},
		"SIGNAL", "core",
	)

	shield.SHIELD_Registry_OperationRegister(operation)
}

func runFlowScenario(execCtx shield.SHIELD_Testing_ExecutionContext) shield.SHIELD_Testing_ScenarioRunResult {
	type scenarioInput struct {
		testSignal signal.Signal
	}
	type scenarioOutput struct{}

	sink := []signal.Signal{}
	sinkMethod := func(input signal.Signal) {
		sink = append(sink, input)
	}

	runConfig := shield.SHIELD_Testing_ScenarioRunConfig{
		MaxIterations: 1,
	}

	scenario := shield.SHIELD_Testing_ScenarioCreate(
		"scenario_basic_flow",
		"Validates a signal can go from: emit -> sink",
		[]shield.SHIELD_Testing_Guard[scenarioInput, scenarioOutput]{
			shield.SHIELD_Testing_GuardCreate(
				"guard",
				scenarioInput{
					testSignal: signal.SignalCreate("ERROR_001"),
				},
				shield.SHIELD_Testing_GuardPolicyPredicate(
					func(_ scenarioOutput) (passed bool, reason string) {
						amountStored := len(sink)
						if amountStored != 1 {
							return false, fmt.Sprintf("expected 1 stored signal, got %d", amountStored)
						}

						storedSignal := sink[0]
						storedID := storedSignal.ID()

						if storedID != "ERROR_001" {
							return false, fmt.Sprintf("expected ID 'ERROR_001', got '%s'", storedID)
						}

						return true, ""
					},
				),
			),
		},
		func(input scenarioInput) (output scenarioOutput, error error) {
			dispatcher := signal.SignalDispatcherCreate()
			signal.SignalDispatcherRegisterSink(dispatcher, sinkMethod)

			signal.SignalDispatcherEmit(dispatcher, input.testSignal)

			return scenarioOutput{}, nil
		},
	)

	return shield.SHIELD_Testing_OperationRunScenario(scenario, execCtx, runConfig)
}
