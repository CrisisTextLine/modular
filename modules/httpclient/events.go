package httpclient

// Event type constants for httpclient module events.
// Following CloudEvents specification reverse domain notation.
const (
	// Client lifecycle events
	EventTypeClientCreated    = "com.modular.httpclient.client.created"
	EventTypeClientConfigured = "com.modular.httpclient.client.configured"

	// Request modifier events
	EventTypeModifierSet     = "com.modular.httpclient.modifier.set"
	EventTypeModifierApplied = "com.modular.httpclient.modifier.applied"

	// Module lifecycle events
	EventTypeModuleStarted = "com.modular.httpclient.module.started"
	EventTypeModuleStopped = "com.modular.httpclient.module.stopped"

	// Configuration events
	EventTypeConfigLoaded = "com.modular.httpclient.config.loaded"
)
