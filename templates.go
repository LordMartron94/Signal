package signal

/*
This file provides some templates for the category manifests.

In order to prevent typos, it is recommended to either:
A) Create constants for the different labels.
B) Create a typed abstraction that resolves to the correct labels if safety is preferred over performance.
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
