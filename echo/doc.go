/*
Package echo bridges `signal.Signal` emissions into the Echo logging pipeline.

[Context]
Register `SignalEchoSinkGet` on a dispatcher alongside other sinks. Echo hooks (console, file, etc.) must be registered before signals are emitted. The sink does not interpret diagnostic categories or payload keys; supply `LogLevelResolver`, `MessageFormatter`, and `FieldsCollector` callbacks from the client to map subsystem-specific signals into Echo log entries.
*/
package echo
