# Signal

Signal is a domain-agnostic diagnostics engine. Producers emit immutable **signals** through a **dispatcher** to one or more **sinks**; each signal carries an id, diagnostic category, optional span trace, optional source location, timestamp, and a payload map. The library does not assume logging, LSP, or CLI—it is an event pipeline you wire to your own consumers.

## Module

### Imports

| Package | Use |
| --- | --- |
| `signal` | Core API: dispatcher, context, signals, builder |
| `signal/rendering` | Optional terminal batch renderer (requires SPLASH) |

### Dependencies

- **Core** — uses `foundation` for optional `*location.Location` on signals (`foundation/location`).
- **Rendering** — `signal/rendering` depends on **SPLASH** (`splash`) for ANSI / true-color terminal output. SPLASH is a separate library; only pull it in if you use the rendering subpackage.

```go
import (
    "signal"

    "foundation/location" // when attaching source locations
)
```

```go
import (
    "signal"
    "signal/rendering"

    "splash"
)
```

## Concepts

- **Diagnostic manifest** — `DiagnosticCategoryManifest` lists category labels and numeric **weights**. Weights drive `SignalDispatcherRegisterFilteredSink`: only signals whose category weight is `>= minWeight` reach that sink. Duplicate labels panic at dispatcher creation.
- **Dispatcher** — `SignalDispatcherCreate` + `SignalDispatcherRegisterSink` / `SignalDispatcherRegisterFilteredSink`. Sinks run **synchronously** on `SignalContextEmit` with a snapshot `Signal` value.
- **Context** — `SignalContextCreate` binds a dispatcher. `SignalContextPushSpan` / `SignalContextPopSpan` build provenance; `SignalContextClone` forks the span stack for concurrent or nested work without sharing mutable span state.
- **Signal** — Created with `SignalContextSignalCreate` or the fluent builder. Read fields via `ID`, `DiagnosticCategory`, `SpanTrace`, `Timestamp`, `Location` / `HasLocation`. Payload access: `SignalPayloadGet`, `SignalPayloadGetAs`, or `SignalPayloadEach` (read-only iteration without exposing the map).
- **Builder** — `SignalContextBuild(ctx, id, category).Payload(...).Location(...).Emit()` for chained construction. **Single-use**: `Build` / `Emit` run a kill-switch that clears internal state; reusing the same builder panics.
- **Location** — Optional `*location.Location` per signal (file/URI-style provenance from Foundation). Pass `nil` for process-wide diagnostics.
- **Rendering** — `signal/rendering` buffers signals from a sink and formats them in one pass (grouped sections + summary) through an injected SPLASH terminal renderer. See [Rendering](#rendering) below.

## Example: emit to a sink

```go
import "signal"

manifest := signal.DiagnosticCategoryManifest{
    {Label: "INFO", Weight: 0},
    {Label: "WARNING", Weight: 10},
    {Label: "ERROR", Weight: 20},
}
dispatcher := signal.SignalDispatcherCreate(manifest)

var received []signal.Signal
signal.SignalDispatcherRegisterSink(dispatcher, "collector", func(sig signal.Signal) {
    received = append(received, sig)
})

ctx := signal.SignalContextCreate(dispatcher)
signal.SignalContextPushSpan(ctx, "parser")

signal.SignalContextBuild(ctx, "ERR_UNEXPECTED_TOKEN", "ERROR").
    Payload("token", "foo").
    Emit()
```

## Example: filtered sink

Only categories with weight ≥ `minWeight` reach the sink (for example `WARNING` at 10 and `ERROR` at 20 when `minWeight` is 10):

```go
signal.SignalDispatcherRegisterFilteredSink(dispatcher, "errors_only", func(sig signal.Signal) {
    // handle matching signals
}, 10)
```

## Example: location

```go
import (
    "signal"

    "foundation/location"
)

loc := location.LocationCreate("file", "", "src/main.go", "", "", nil)

signal.SignalContextBuild(ctx, "ERR_UNDEFINED", "ERROR").
    Location(&loc).
    Payload("symbol", "foo").
    Emit()
```

## Rendering

`signal/rendering` is optional. It adapts **SPLASH** for batched CLI-style output:

1. In your application, create a SPLASH palette and `splash.SPLASH_Rendering_TerminalRenderer` (see the SPLASH module documentation).
2. `rendering.SignalRendererCreate(renderer, grouping, metaIntent)` — sets indent width to 2 on the SPLASH renderer.
3. Register `rendering.SignalRendererSinkGet(renderer)` on the Signal dispatcher.
4. Emit signals as usual.
5. `rendering.SignalRendererRender(renderer)` returns the formatted string and clears the capture buffer.

`GroupingConfiguration` supplies:

- `ExtractKey` — bucket key per signal (e.g. category string or file path from `Location`).
- `PriorityOrder` — which buckets appear first in the body and summary.
- `ResolveIntent` — SPLASH palette slot for group headers, summary pills, and per-line category color.

```go
import (
    "fmt"

    "signal"
    "signal/rendering"

    "splash"
)

grouping := rendering.GroupingConfiguration{
    ExtractKey: func(sig signal.Signal) string {
        return string(sig.DiagnosticCategory())
    },
    ResolveIntent: func(key string) int {
        switch key {
        case "ERROR":
            return 0 // must match a registered SPLASH palette key
        default:
            return 1
        }
    },
    PriorityOrder: []string{"ERROR", "WARNING", "INFO"},
}

palette := /* splash.SPLASH_Rendering_TerminalPaletteBuilderCreate(...).Build() */
metaIntent := 2 // palette key for trace/payload labels

splashRenderer := splash.SPLASH_Rendering_TerminalRendererCreate(
    splash.SPLASH_Rendering_TerminalColorModeAnsi16,
    palette,
)
buf := rendering.SignalRendererCreate(splashRenderer, grouping, metaIntent)
signal.SignalDispatcherRegisterSink(dispatcher, "cli", rendering.SignalRendererSinkGet(buf))

// ... emit signals ...

fmt.Print(rendering.SignalRendererRender(buf))
```
