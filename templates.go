package signal

/*
DiagnosticCategoryManifest_Logging provides a standard logging-oriented category manifest.

[Context]
Use this when integrating Signal as a generic application logging pipeline.
It defines conventional severities from TRACE through FATAL with increasing weights.
*/
var DiagnosticCategoryManifest_Logging = DiagnosticCategoryManifest{
	{
		Label:  "TRACE",
		Weight: 0,
	},
	{
		Label:  "DEBUG",
		Weight: 10,
	},
	{
		Label:  "INFO",
		Weight: 20,
	},
	{
		Label:  "NOTICE",
		Weight: 25,
	},
	{
		Label:  "WARNING",
		Weight: 30,
	},
	{
		Label:  "ERROR",
		Weight: 40,
	},
	{
		Label:  "FATAL",
		Weight: 50,
	},
}

/*
DiagnosticCategoryManifest_LSP provides a language-server-style diagnostics category manifest.

[Context]
Use this for editor or LSP diagnostic flows where categories map to hint/information/warning/error severities.
*/
var DiagnosticCategoryManifest_LSP = DiagnosticCategoryManifest{
	{
		Label:  "HINT",
		Weight: 0,
	},
	{
		Label:  "INFORMATION",
		Weight: 10,
	},
	{
		Label:  "WARNING",
		Weight: 20,
	},
	{
		Label:  "ERROR",
		Weight: 30,
	},
}
