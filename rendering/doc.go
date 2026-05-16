/*
Package rendering turns buffered `signal.Signal` values into terminal-oriented text via SPLASH.

[Context]
This subpackage sits beside the core `signal` API: producers still emit through a dispatcher, but a `SignalRenderer` sink accumulates signals and `SignalRendererRender` formats them in one pass (grouped headers, per-signal detail, and a summary footer). Coloring and indentation are delegated to an injected `splash.SPLASH_Rendering_TerminalRenderer`; grouping keys and palette intents are supplied by `GroupingConfiguration`.

[Model]
Register `SignalRendererSinkGet` on a dispatcher to capture emissions, then call `SignalRendererRender` when the batch is complete. Render drains the internal signal buffer and returns the SPLASH renderer's string; an empty capture yields an empty string without writing to SPLASH.

[Thread Safety]
`SignalRenderer` synchronizes its signal buffer; the returned sink and `SignalRendererRender` may be used from concurrent emitters as long as they target the same renderer instance.
*/
package rendering
