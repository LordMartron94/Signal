package rendering

import (
	"fmt"
	"foundation/location"
	"signal"
	"splash"
	"strings"
	"sync"
)

/*
GroupKeyExtractor derives the bucket key used to group signals for headers and summary pills.

[Context]
Typical keys are diagnostic category labels, file paths, or other provenance strings extracted from the signal (for example via span trace or payload).
*/
type GroupKeyExtractor func(sig signal.Signal) string

/*
IntentResolver maps a group or category key to a palette color key for SPLASH.

[Returns]
Returns an integer in `[0, paletteColorAmount)` understood by the `splash.SPLASH_Rendering_TerminalRenderer` palette passed at creation time. When a key is unknown, return a safe default intent registered on the palette.
*/
type IntentResolver func(groupKey string) int

/*
GroupingConfiguration controls how buffered signals are partitioned, ordered, and colored during render.

[Context]
`ExtractKey` assigns each signal to a bucket. `PriorityOrder` lists bucket keys emitted first (both in the body and in the summary); any remaining buckets follow in map iteration order. `ResolveIntent` selects the palette slot for group headers, summary pills, and per-signal category lines (category string is passed for individual signals).
*/
type GroupingConfiguration struct {
	ExtractKey    GroupKeyExtractor
	ResolveIntent IntentResolver
	PriorityOrder []string
}

/*
LocationFormatter turns a signal's `*location.Location` into a suffix printed on the same line as the signal id.

[Context]
The renderer calls this only when the signal reports `HasLocation()` and the formatter is non-nil. Return an empty string to omit location text (for example when coordinates are absent). Typical output is human-readable provenance such as ` at line 42` or a path fragment; include leading spacing if you want separation from the id.

[Parameters]
`loc` is non-nil when invoked.

[Side Effects]
Pure function. No side effects.
*/
type LocationFormatter func(loc *location.Location) string

/*
SignalRenderer buffers signals from a sink and formats them through SPLASH on demand.

[Context]
Create with `SignalRendererCreate`, register the sink from `SignalRendererSinkGet` on a `signal.SignalDispatcher`, emit signals, then call `SignalRendererRender` to produce CLI-style output. Indent width on the injected SPLASH renderer is set to two spaces at creation. Optional location suffixes on each signal line come from the `LocationFormatter` supplied at creation.

[Thread Safety]
Safe for concurrent appends via the sink and a concurrent `SignalRendererRender` on the same instance; both paths lock the internal buffer.
*/
type SignalRenderer struct {
	mutex          sync.Mutex
	renderer       *splash.SPLASH_Rendering_TerminalRenderer
	buffer         []signal.Signal
	grouping       GroupingConfiguration
	formatLocation LocationFormatter
	metaIntent     int
}

/*
SignalRendererCreate constructs a renderer bound to a SPLASH terminal renderer and grouping policy.

[Parameters]
`renderer` must already be created with a palette sized for every intent returned by `grouping.ResolveIntent` and for `metaIntent` (used for trace and payload labels). `metaIntent` is the palette key for structural/meta lines such as `Trace:` and `Payload:`.
`formatLocation` formats optional source locations on the signal header line (`[CATEGORY] id` + formatter suffix). Pass `nil` to never print location text even when signals carry a location.

[Side Effects]
Sets `renderer` indent width to 2. Allocates an empty signal buffer.
*/
func SignalRendererCreate(
	renderer *splash.SPLASH_Rendering_TerminalRenderer,
	grouping GroupingConfiguration,
	formatLocation LocationFormatter,
	metaIntent int,
) *SignalRenderer {
	splash.SPLASH_Rendering_TerminalRendererSetIndentWidth(renderer, 2)

	return &SignalRenderer{
		renderer:       renderer,
		buffer:         make([]signal.Signal, 0),
		grouping:       grouping,
		formatLocation: formatLocation,
		metaIntent:     metaIntent,
	}
}

/*
SignalRendererSinkGet returns a `signal.SignalSink` that appends each emitted signal to this renderer's buffer.

[Returns]
Returns a sink suitable for `signal.SignalDispatcherRegisterSink`. Emissions are copied into internal storage; the sink does not render immediately.

[Side Effects]
Mutates the renderer's buffer under lock on each invocation.
*/
func SignalRendererSinkGet(s *SignalRenderer) signal.SignalSink {
	return func(sig signal.Signal) {
		s.mutex.Lock()
		defer s.mutex.Unlock()
		s.buffer = append(s.buffer, sig)
	}
}

/*
SignalRendererRender formats all buffered signals, clears the buffer, and returns the SPLASH output string.

[Returns]
Returns an empty string when nothing was buffered. Otherwise returns grouped sections (priority keys first, then remaining buckets), a `=== SUMMARY ===` footer with count pills, and trailing newline spacing as produced by SPLASH.

[Context]
Per signal, output includes category-colored `[CATEGORY] id`, an optional location suffix from `formatLocation` when the signal has a location and the formatter returns non-empty text, optional indented span trace, and indented payload key/value lines via `signal.SignalPayloadEach`. Group headers use `ResolveIntent` on the bucket key; category lines use `ResolveIntent` on the diagnostic category string.

[Side Effects]
Clears the internal signal buffer after formatting. Invokes SPLASH buffer operations and `SPLASH_Rendering_TerminalRendererRender`, which resets the SPLASH string buffer while preserving SPLASH indent state.
*/
func SignalRendererRender(s *SignalRenderer) string {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if len(s.buffer) == 0 {
		return ""
	}

	buckets := make(map[string][]signal.Signal)
	counts := make(map[string]int)

	for _, sig := range s.buffer {
		key := s.grouping.ExtractKey(sig)
		buckets[key] = append(buckets[key], sig)
		counts[key]++
	}

	renderer := s.renderer

	for _, key := range s.grouping.PriorityOrder {
		signals, exists := buckets[key]
		if !exists || len(signals) == 0 {
			continue
		}

		renderGroupHeader(s, key, len(signals))
		for _, sig := range signals {
			renderSignal(s, sig)
		}

		delete(buckets, key)
	}

	for key, signals := range buckets {
		if len(signals) == 0 {
			continue
		}
		renderGroupHeader(s, key, len(signals))
		for _, sig := range signals {
			renderSignal(s, sig)
		}
	}

	splash.SPLASH_Rendering_TerminalRendererBufferLineBreak(renderer)
	splash.SPLASH_Rendering_TerminalRendererBufferContent(renderer, "=== SUMMARY ===")
	splash.SPLASH_Rendering_TerminalRendererBufferLineBreak(renderer)

	for _, key := range s.grouping.PriorityOrder {
		if count, exists := counts[key]; exists {
			renderSummaryPill(s, key, count)
		}
		delete(counts, key)
	}

	for key, count := range counts {
		renderSummaryPill(s, key, count)
	}

	splash.SPLASH_Rendering_TerminalRendererBufferLineBreak(renderer)

	s.buffer = nil
	return splash.SPLASH_Rendering_TerminalRendererRender(s.renderer)
}

func renderGroupHeader(s *SignalRenderer, key string, count int) {
	intent := s.grouping.ResolveIntent(key)

	headerText := fmt.Sprintf("--- %s (%d) ---", key, count)
	splash.SPLASH_Rendering_TerminalRendererBufferColoredContent(s.renderer, headerText, intent)
	splash.SPLASH_Rendering_TerminalRendererBufferLineBreak(s.renderer)
	splash.SPLASH_Rendering_TerminalRendererBufferLineBreak(s.renderer)
}

func renderSummaryPill(s *SignalRenderer, key string, count int) {
	intent := s.grouping.ResolveIntent(key)

	splash.SPLASH_Rendering_TerminalRendererBufferContent(s.renderer, "[ ")
	splash.SPLASH_Rendering_TerminalRendererBufferContent(s.renderer, fmt.Sprintf("%d ", count))
	splash.SPLASH_Rendering_TerminalRendererBufferColoredContent(s.renderer, key, intent)
	splash.SPLASH_Rendering_TerminalRendererBufferContent(s.renderer, " ] ")
}

func renderSignal(s *SignalRenderer, sig signal.Signal) {
	cat := string(sig.DiagnosticCategory())

	intent := s.grouping.ResolveIntent(cat)

	renderer := s.renderer

	splash.SPLASH_Rendering_TerminalRendererBufferContent(renderer, "[")
	splash.SPLASH_Rendering_TerminalRendererBufferColoredContent(renderer, cat, intent)
	splash.SPLASH_Rendering_TerminalRendererBufferContent(renderer, "] ")
	splash.SPLASH_Rendering_TerminalRendererBufferContent(renderer, sig.ID())

	if sig.HasLocation() && s.formatLocation != nil {
		loc := sig.Location()
		if loc != nil {
			coordText := s.formatLocation(loc)
			if coordText != "" {
				splash.SPLASH_Rendering_TerminalRendererBufferContent(renderer, coordText)
			}
		}
	}

	splash.SPLASH_Rendering_TerminalRendererBufferLineBreak(renderer)

	spans := sig.SpanTrace()
	if len(spans) > 0 {
		splash.SPLASH_Rendering_TerminalRendererIndent(renderer)
		splash.SPLASH_Rendering_TerminalRendererBufferColoredContent(renderer, "Trace: ", s.metaIntent)
		splash.SPLASH_Rendering_TerminalRendererBufferContent(renderer, strings.Join(spans, " > "))
		splash.SPLASH_Rendering_TerminalRendererBufferLineBreak(renderer)
		splash.SPLASH_Rendering_TerminalRendererDedent(renderer)
	}

	renderPayload(s, sig)

	splash.SPLASH_Rendering_TerminalRendererBufferLineBreak(renderer)
}

func renderPayload(s *SignalRenderer, sig signal.Signal) {
	hasPayload := false
	renderer := s.renderer

	signal.SignalPayloadEach(&sig, func(key string, value any) {
		if !hasPayload {
			splash.SPLASH_Rendering_TerminalRendererIndent(renderer)
			splash.SPLASH_Rendering_TerminalRendererBufferColoredContent(renderer, "Payload:", s.metaIntent)
			splash.SPLASH_Rendering_TerminalRendererBufferLineBreak(renderer)
			splash.SPLASH_Rendering_TerminalRendererIndent(renderer)
			hasPayload = true
		}

		splash.SPLASH_Rendering_TerminalRendererBufferColoredContent(renderer, key, s.metaIntent)
		splash.SPLASH_Rendering_TerminalRendererBufferContent(renderer, fmt.Sprintf("=%v", value))
		splash.SPLASH_Rendering_TerminalRendererBufferLineBreak(renderer)
	})

	if hasPayload {
		splash.SPLASH_Rendering_TerminalRendererDedent(renderer)
		splash.SPLASH_Rendering_TerminalRendererDedent(renderer)
	}
}
