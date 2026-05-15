package rendering

import (
	"fmt"
	"signal"
	"strings"
	"sync"
)

const (
	RenderColorNone = iota + 0
)

type SignalRenderer struct {
	mutex     sync.Mutex
	colorMode int
	buffer    []signal.Signal
}

func SignalRendererCreate(colorMode int) *SignalRenderer {
	return &SignalRenderer{
		colorMode: colorMode,
		buffer:    make([]signal.Signal, 0),
	}
}

/*
SignalRendererSinkGet returns a closure satisfying the SignalSink signature.
This bridges the stateless pipeline emission with the stateful renderer buffer.
*/
func SignalRendererSinkGet(s *SignalRenderer) signal.SignalSink {
	return func(sig signal.Signal) {
		s.mutex.Lock()
		defer s.mutex.Unlock()
		s.buffer = append(s.buffer, sig)
	}
}

/*
SignalRendererRender flushes the captured signals into a formatted string.
It resets the internal buffer to prevent memory leaks on subsequent calls.
*/
func SignalRendererRender(s *SignalRenderer) string {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if len(s.buffer) == 0 {
		return ""
	}

	builder := &strings.Builder{}

	for _, sig := range s.buffer {
		builder.WriteString(fmt.Sprintf("[%s] %s\n", sig.DiagnosticCategory(), sig.ID()))

		spans := sig.SpanTrace()
		if len(spans) > 0 {
			builder.WriteString("  Trace: ")
			builder.WriteString(strings.Join(spans, " > "))
			builder.WriteString("\n")
		}

		hasPayload := false
		signal.SignalPayloadEach(&sig, func(key string, value any) {
			if !hasPayload {
				builder.WriteString("  Payload:\n")
				hasPayload = true
			}
			builder.WriteString(fmt.Sprintf("    %s=%v\n", key, value))
		})

		builder.WriteString("\n")
	}

	s.buffer = nil

	return builder.String()
}
