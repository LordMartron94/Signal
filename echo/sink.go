package echo

import (
	"echo"
	"essence"
	"signal"
)

/*
LogLevelResolver maps a signal to the Echo log level used when forwarding it.
*/
type LogLevelResolver func(sig signal.Signal) echo.LogLevel

/*
MessageFormatter produces the primary log message string for a signal.
*/
type MessageFormatter func(sig signal.Signal) string

/*
FieldsCollector builds structured fields attached to the Echo log entry.
*/
type FieldsCollector func(sig signal.Signal) map[string]interface{}

/*
SinkConfiguration controls how signals are translated into Echo log entries.

[Context]
All interpretation of diagnostic categories, subsystem payload keys, and severity semantics belongs in the client callbacks. This package only forwards resolved values to Echo.
*/
type SinkConfiguration struct {
	SystemID        essence.UUID
	ResolveLogLevel LogLevelResolver
	FormatMessage   MessageFormatter
	CollectFields   FieldsCollector
}

/*
SignalEchoSinkGet returns a sink that forwards each signal to Echo using the client-supplied resolvers.
*/
func SignalEchoSinkGet(config SinkConfiguration) signal.SignalSink {
	if config.ResolveLogLevel == nil {
		panic("signal/echo: SinkConfiguration.ResolveLogLevel must not be nil")
	}

	return func(sig signal.Signal) {
		level := config.ResolveLogLevel(sig)

		message := sig.ID()
		if config.FormatMessage != nil {
			message = config.FormatMessage(sig)
		}

		var fields map[string]interface{}
		if config.CollectFields != nil {
			fields = config.CollectFields(sig)
		}

		event := echo.On(config.SystemID)
		if fields != nil {
			event = event.Fields(fields)
		}

		switch level {
		case echo.TRACE:
			event.Trace(message)
		case echo.DEBUG:
			event.Debug(message)
		case echo.INFO:
			event.Info(message)
		case echo.NOTICE:
			event.Notice(message)
		case echo.WARNING:
			event.Warning(message)
		case echo.ERROR:
			event.Error(message)
		case echo.CRITICAL:
			event.Critical(message)
		default:
			event.Info(message)
		}
	}
}
