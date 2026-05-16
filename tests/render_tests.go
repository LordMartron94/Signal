package tests

import (
	"fmt"
	"foundation/location"
	"signal"
	"signal/rendering"
	"splash"
	"testing"
)

const (
	IntentCategoryInfo = iota
	IntentCategoryWarning
	IntentCategoryError
	IntentDefault
	IntentMeta
	intentCount
)

var defaultRenderingIntentMap = map[string]int{
	"INFO":    IntentCategoryInfo,
	"WARNING": IntentCategoryWarning,
	"ERROR":   IntentCategoryError,
}

var categoryGroupingStrategy = rendering.GroupingConfiguration{
	ExtractKey: func(sig signal.Signal) string {
		return string(sig.DiagnosticCategory())
	},
	ResolveIntent: func(groupKey string) int {
		intent, exists := defaultRenderingIntentMap[groupKey]
		if !exists {
			return IntentDefault
		}
		return intent
	},
	PriorityOrder: []string{"ERROR", "WARNING", "INFO"},
}

var fileGroupingStrategy = rendering.GroupingConfiguration{
	ExtractKey: func(sig signal.Signal) string {
		if sig.HasLocation() && sig.Location().Path() != "" {
			return sig.Location().Path()
		}
		return "Global Diagnostics"
	},
	ResolveIntent: func(key string) int {
		if intent, exists := defaultRenderingIntentMap[key]; exists {
			return intent
		}

		return IntentMeta
	},
	PriorityOrder: []string{"Global Diagnostics"},
}

var clientLocationFormatter = func(loc *location.Location) string {
	line, errL := location.LocationCoordinateGetAs[int](*loc, "line")
	if errL == nil && line > 0 {
		return fmt.Sprintf(" at line %d", line)
	}

	return ""
}

func SignalTestRenderer(t *testing.T) {
	// 1. Client builds the visual Palette
	paletteBuilder := splash.SPLASH_Rendering_TerminalPaletteBuilderCreate(int(intentCount))

	paletteBuilder.Register(IntentCategoryError, splash.SPLASH_Rendering_TerminalColorAnsi16_Red, splash.SPLASH_Rendering_TerminalTrueColor(231, 76, 60))
	paletteBuilder.Register(IntentCategoryWarning, splash.SPLASH_Rendering_TerminalColorAnsi16_Yellow, splash.SPLASH_Rendering_TerminalTrueColor(241, 196, 15))
	paletteBuilder.Register(IntentCategoryInfo, splash.SPLASH_Rendering_TerminalColorAnsi16_Cyan, splash.SPLASH_Rendering_TerminalTrueColor(52, 152, 219))
	paletteBuilder.Register(IntentDefault, splash.SPLASH_Rendering_TerminalColorAnsi16_BrightBlack, splash.SPLASH_Rendering_TerminalTrueColor(127, 140, 141))
	paletteBuilder.Register(IntentMeta, splash.SPLASH_Rendering_TerminalColorAnsi16_BrightBlack, splash.SPLASH_Rendering_TerminalTrueColor(127, 140, 141))

	palette := paletteBuilder.Build()

	// 2. Client initializes Splash engines
	splashNone := splash.SPLASH_Rendering_TerminalRendererCreate(splash.SPLASH_Rendering_TerminalColorModeNone, palette)
	splashANSI := splash.SPLASH_Rendering_TerminalRendererCreate(splash.SPLASH_Rendering_TerminalColorModeAnsi16, palette)
	splashTrue := splash.SPLASH_Rendering_TerminalRendererCreate(splash.SPLASH_Rendering_TerminalColorModeTrueColor, palette)

	// 3. Client initializes the Adapters
	// ANSI mode uses Category grouping, TrueColor uses File grouping to prove the divergence.
	rendererNone := rendering.SignalRendererCreate(splashNone, categoryGroupingStrategy, clientLocationFormatter, nil, IntentMeta)
	rendererANSI := rendering.SignalRendererCreate(splashANSI, categoryGroupingStrategy, clientLocationFormatter, nil, IntentMeta)
	rendererTrue := rendering.SignalRendererCreate(splashTrue, fileGroupingStrategy, clientLocationFormatter, nil, IntentMeta)

	// 4. Setup the Dispatcher
	manifest := signal.DiagnosticCategoryManifest{
		{Label: "INFO", Weight: 0},
		{Label: "WARNING", Weight: 10},
		{Label: "ERROR", Weight: 20},
	}
	dispatcher := signal.SignalDispatcherCreate(manifest)

	// 5. Register them as distinct sinks
	signal.SignalDispatcherRegisterSink(dispatcher, "sink_none", rendering.SignalRendererSinkGet(rendererNone))
	signal.SignalDispatcherRegisterSink(dispatcher, "sink_ansi", rendering.SignalRendererSinkGet(rendererANSI))
	signal.SignalDispatcherRegisterSink(dispatcher, "sink_true", rendering.SignalRendererSinkGet(rendererTrue))

	// 6. Build a nested context
	ctx := signal.SignalContextCreate(dispatcher)
	signal.SignalContextPushSpan(ctx, "Compiler")
	signal.SignalContextPushSpan(ctx, "Lexer")

	// Dummy locations to test the file extractor
	locMain := location.LocationCreate("file", "", "src/main.go", "", "", nil)
	locParser := location.LocationCreate("file", "", "src/parser.go", "", "", nil)

	// 7. Emit out of order to prove grouping logic

	// First: An INFO signal (Global, no location)
	signal.SignalContextBuild(ctx, "INFO_FILE_LOADED", "INFO").
		Payload("bytes", 4096).
		Emit()

	signal.SignalContextPopSpan(ctx)
	signal.SignalContextPushSpan(ctx, "Parser")

	// Second: A WARNING signal tied to src/main.go
	signal.SignalContextBuild(ctx, "WARN_UNUSED_VAR", "WARNING").
		Location(&locMain).
		Payload("variable", "index").
		Payload("line", 14).
		Emit()

	// Third: An ERROR signal tied to src/parser.go
	signal.SignalContextBuild(ctx, "ERR_SYNTAX", "ERROR").
		Location(&locParser).
		Payload("expected", "semicolon").
		Payload("line", 15).
		Emit()

	// Fourth: Another ERROR signal tied to src/main.go
	signal.SignalContextBuild(ctx, "ERR_UNDEFINED_FUNC", "ERROR").
		Location(&locMain).
		Payload("func", "calculate_offset").
		Payload("line", 88).
		Emit()

	// 8. Visual Output Execution
	fmt.Println("========================================")
	fmt.Println(" MODE: NONE (Grouped by Category)")
	fmt.Println("========================================")
	fmt.Print(rendering.SignalRendererRender(rendererNone))

	fmt.Println("========================================")
	fmt.Println(" MODE: ANSI 16 (Grouped by Category)")
	fmt.Println("========================================")
	fmt.Print(rendering.SignalRendererRender(rendererANSI))

	fmt.Println("========================================")
	fmt.Println(" MODE: TRUE COLOR (Grouped by File)")
	fmt.Println("========================================")
	fmt.Print(rendering.SignalRendererRender(rendererTrue))
}
