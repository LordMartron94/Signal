package tests

import (
	"fmt"
	"shield"
	"signal"
)

func SignalMainUnit(order int) shield.Unit {
	unit := shield.UnitCreate(order, "Signal")
	shield.UnitSetDescription(unit, "Validates the entire Signal framework functions as expected.")

	storeUnit := getStoreUnit()

	shield.UnitRegisterSubUnits(unit, storeUnit)

	return *unit
}

type SignalTestOutput struct {
	panicked     bool
	panicMessage string
}

func getStoreUnit() shield.Unit {
	storeUnit := shield.UnitCreate(0, "Signal Store")
	shield.UnitSetDescription(storeUnit, "Validates the signal store system works as intended.")

	setup := func() {

	}

	teardown := func() {

	}

	shield.UnitSetSetupAndTeardown(storeUnit, setup, teardown)

	creationAtom := shield.AtomCreate(0, "creation", func(_ struct{}) (output SignalTestOutput) {
		defer func() {
			if r := recover(); r != nil {
				output.panicked = true
				output.panicMessage = fmt.Sprintf("%v", r)
			}
		}()

		signal.SignalStoreCreate()

		return output
	})

	shield.AtomRegisterCase(creationAtom, shield.CaseCreate("default", struct{}{}, func(output SignalTestOutput) shield.AtomResult {
		if output.panicked {
			return *shield.AtomResultFailureCreate(fmt.Sprintf("Failed: unexpected panic during creation: %s", output.panicMessage))
		}

		return *shield.AtomResultSuccessCreate()
	}))

	shield.UnitRegisterAtom(storeUnit, creationAtom)

	return *storeUnit
}
