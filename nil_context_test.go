package signal_test

import (
	"testing"

	"signal"
)

func TestSignalContextNilSpanNoOp(t *testing.T) {
	signal.SignalContextPushSpan(nil, "span")
	signal.SignalContextPopSpan(nil)
}

func TestSignalContextNoOpDoesNotDispatch(t *testing.T) {
	var emitted int

	dispatcher := signal.SignalDispatcherCreate(signal.DiagnosticCategoryManifest{
		{Label: "DEBUG", Weight: 0},
	})
	signal.SignalDispatcherRegisterSink(dispatcher, "test", func(signal.Signal) {
		emitted++
	})

	ctx := signal.SignalContextCreateNoOp(dispatcher)
	signal.SignalContextBuild(ctx, "TestSignal", "DEBUG").Emit()

	if emitted != 0 {
		t.Fatalf("expected 0 sink invocations, got %d", emitted)
	}
}
