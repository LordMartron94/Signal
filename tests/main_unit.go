package tests

import (
	"fmt"
	"shield"
	"signal"
)

type caseValidator func(output signalTestOutput) shield.AtomResult

type signalTestOutput struct {
	panicked     bool
	panicMessage string

	error error
}

func SignalMainUnit(order int) shield.Unit {
	unit := shield.UnitCreate(order, "Signal")
	shield.UnitSetDescription(unit, "Validates the entire Signal framework functions as expected.")

	storeUnit := getStoreUnit()

	shield.UnitRegisterSubUnits(unit, storeUnit)

	return *unit
}

// ------------------------------------------------------------------ UNITS

func getStoreUnit() shield.Unit {
	storeUnit := shield.UnitCreate(0, "Signal Store")
	shield.UnitSetDescription(storeUnit, "Validates the signal store system works as intended.")

	setup := func() {

	}

	teardown := func() {

	}

	shield.UnitSetSetupAndTeardown(storeUnit, setup, teardown)
	creationAtom := getCreationAtom(0)
	storeAtom := getStoreAtom(1)

	shield.UnitRegisterAtom(storeUnit, creationAtom)
	shield.UnitRegisterAtom(storeUnit, storeAtom)

	return *storeUnit
}

// ------ CREATION ------

func getCreationAtom(order int) *shield.Atom[struct{}, signalTestOutput] {
	creationAtom := shield.AtomCreate(order, "creation", func(_ struct{}) (output signalTestOutput) {
		defer catchOutputPanic(&output)

		signal.SignalStoreCreate()

		return output
	})
	shield.AtomSetDescription(creationAtom, "Validates creation does not panic.")

	panicValidator := getCaseValidator(
		func(panicMessage string) string {
			return fmt.Sprintf("unexpected panic during creation: %s", panicMessage)
		},
		func(errorMessage string) string {
			return fmt.Sprintf("unexpected error during creation: %s", errorMessage)
		},
		nil,
	)

	shield.AtomRegisterCase(creationAtom, shield.CaseCreate(
		"default", struct{}{},
		func(output signalTestOutput) shield.AtomResult {
			return panicValidator(output)
		},
	))

	return creationAtom
}

// ------ STORING ------

func getStoreAtom(order int) *shield.Atom[signal.Signal, signalTestOutput] {
	var store *signal.SignalStore = nil

	runner := func(in signal.Signal) (output signalTestOutput) {
		defer catchOutputPanic(&output)

		err := signal.SignalStoreStore(store, in)
		output.error = err

		return output
	}

	setup := func() {
		store = signal.SignalStoreCreate()
	}
	teardown := func() {
		store = nil
	}

	storeAtom := shield.AtomCreate(order, "store", runner)
	shield.AtomSetDescription(storeAtom, "Validates the signal store storing functionality works correctly.")
	shield.AtomSetSetupAndTeardown(storeAtom, setup, teardown)

	mainValidator := getCaseValidator(
		func(panicMessage string) string {
			return fmt.Sprintf("unexpected panic during store operation: %s", panicMessage)
		},
		func(errorMessage string) string {
			return fmt.Sprintf("unexpected error during store operation: %s", errorMessage)
		},
		nil,
	)

	shield.AtomRegisterCase(storeAtom, shield.CaseCreate(
		"default", *signal.SignalCreate(),
		func(output signalTestOutput) shield.AtomResult {
			return mainValidator(output)
		},
	))

	return storeAtom
}

// ------------------------------------------------------------------ VALIDATORS

func getCaseValidator(
	panicMessageFormatter func(panicMessage string) string,
	errorMessageFormatter func(panicMessage string) string,
	subValidator func(output signalTestOutput) shield.AtomResult,
) caseValidator {
	return func(output signalTestOutput) shield.AtomResult {
		if output.panicked {
			return *shield.AtomResultFailureCreate(panicMessageFormatter(output.panicMessage))
		}

		if output.error != nil {
			return *shield.AtomResultFailureCreate(errorMessageFormatter(output.error.Error()))
		}

		if subValidator != nil {
			return subValidator(output)
		}

		return *shield.AtomResultSuccessCreate()
	}
}

// ------------------------------------------------------------------ PRIVATE HELPERS

func catchOutputPanic(output *signalTestOutput) {
	if r := recover(); r != nil {
		output.panicked = true
		output.panicMessage = fmt.Sprintf("%v", r)
	}
}
