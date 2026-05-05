package tests

import (
	"fmt"
	"shield"
	"signal"
)

type storeInput struct {
	signals []signal.Signal
}

func init() {
	creationScenario := buildStoreCreationScenario()
	storeScenario := buildStoreScenario()
	runCfg := shield.SHIELD_Testing_ScenarioRunConfig{
		MaxIterations: 1,
	}

	operation := shield.SHIELD_Testing_OperationCreateStateless(
		"signal_store_operation",
		"Validates Signal store lifecycle behavior, including creation and storing multiple signals.",
		func(_ struct{}, execCtx shield.SHIELD_Testing_ExecutionContext) []shield.SHIELD_Testing_ScenarioRunResult {
			return []shield.SHIELD_Testing_ScenarioRunResult{
				shield.SHIELD_Testing_OperationRunScenario(creationScenario, execCtx, runCfg),
				shield.SHIELD_Testing_OperationRunScenario(storeScenario, execCtx, runCfg),
			}
		},
		"SIGNAL", "Store",
	)
	shield.SHIELD_Registry_OperationRegister(operation)
}

func buildStoreCreationScenario() shield.SHIELD_Testing_Scenario[struct{}, bool] {
	guard := shield.SHIELD_Testing_GuardCreate(
		"store_creation_does_not_panic",
		struct{}{},
		shield.SHIELD_Testing_GuardPolicyMustEqual(
			func(a, b bool) bool { return a == b },
			func(item bool) string { return fmt.Sprintf("%t", item) },
			true,
		),
	)

	scenario := shield.SHIELD_Testing_ScenarioCreate(
		"signal_store_create",
		"Ensures SignalStoreCreate returns a valid store instance without panicking.",
		[]shield.SHIELD_Testing_Guard[struct{}, bool]{guard},
		func(_ struct{}) (bool, error) {
			store := signal.SignalStoreCreate()
			return store != nil, nil
		},
	)
	return scenario
}

func buildStoreScenario() shield.SHIELD_Testing_Scenario[storeInput, int] {
	guardSingle := shield.SHIELD_Testing_GuardCreate(
		"store_single_signal",
		storeInput{
			signals: []signal.Signal{*signal.SignalCreate()},
		},
		shield.SHIELD_Testing_GuardPolicyMustEqual(
			func(a, b int) bool { return a == b },
			func(item int) string { return fmt.Sprintf("%d", item) },
			1,
		),
	)
	guardDouble := shield.SHIELD_Testing_GuardCreate(
		"store_double_signal",
		storeInput{
			signals: []signal.Signal{*signal.SignalCreate(), *signal.SignalCreate()},
		},
		shield.SHIELD_Testing_GuardPolicyMustEqual(
			func(a, b int) bool { return a == b },
			func(item int) string { return fmt.Sprintf("%d", item) },
			2,
		),
	)

	scenario := shield.SHIELD_Testing_ScenarioCreate(
		"signal_store_store",
		"Verifies SignalStoreStore accepts single and multiple signals and reports the expected stored count.",
		[]shield.SHIELD_Testing_Guard[storeInput, int]{guardSingle, guardDouble},
		func(input storeInput) (int, error) {
			store := signal.SignalStoreCreate()
			for _, signalToAdd := range input.signals {
				if err := signal.SignalStoreStore(store, signalToAdd); err != nil {
					return 0, err
				}
			}
			return signal.SignalStoreStoredAmountGet(store), nil
		},
	)
	return scenario
}
