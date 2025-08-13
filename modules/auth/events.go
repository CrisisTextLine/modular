package auth

// Event type constants for auth module events.
// Following CloudEvents specification reverse domain notation.
const (
	// Authentication events
	EventTypeAuthAttempt     = "com.modular.auth.attempt"
	EventTypeAuthSuccess     = "com.modular.auth.success"
	EventTypeAuthFailure     = "com.modular.auth.failure"
	EventTypeAuthLogout      = "com.modular.auth.logout"
	
	// Token events
	EventTypeTokenGenerated  = "com.modular.auth.token.generated"
	EventTypeTokenValidated  = "com.modular.auth.token.validated"
	EventTypeTokenExpired    = "com.modular.auth.token.expired"
	EventTypeTokenRefreshed  = "com.modular.auth.token.refreshed"
	
	// Session events
	EventTypeSessionCreated  = "com.modular.auth.session.created"
	EventTypeSessionAccessed = "com.modular.auth.session.accessed"
	EventTypeSessionExpired  = "com.modular.auth.session.expired"
	EventTypeSessionDestroyed = "com.modular.auth.session.destroyed"
	
	// User management events
	EventTypeUserRegistered  = "com.modular.auth.user.registered"
	EventTypeUserUpdated     = "com.modular.auth.user.updated"
	EventTypeUserLocked      = "com.modular.auth.user.locked"
	EventTypeUserUnlocked    = "com.modular.auth.user.unlocked"
	
	// Password events
	EventTypePasswordChanged = "com.modular.auth.password.changed"
	EventTypePasswordReset   = "com.modular.auth.password.reset"
	
	// OAuth2 events
	EventTypeOAuth2AuthURL   = "com.modular.auth.oauth2.auth_url"
	EventTypeOAuth2Callback  = "com.modular.auth.oauth2.callback"
	EventTypeOAuth2Exchange  = "com.modular.auth.oauth2.exchange"
)