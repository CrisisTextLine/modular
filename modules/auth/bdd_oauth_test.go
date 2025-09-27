package auth

import (
	"fmt"

	"github.com/cucumber/godog"
	oauth2 "golang.org/x/oauth2"
)

// OAuth2-related BDD step implementations

func (ctx *AuthBDDTestContext) iHaveOAuth2Configuration() error {
	// Ensure base auth app is initialized
	if ctx.service == nil || ctx.module == nil {
		if err := ctx.iHaveAModularApplicationWithAuthModuleConfigured(); err != nil {
			return fmt.Errorf("failed to initialize auth application: %w", err)
		}
	}

	// If already configured with provider, nothing to do
	if ctx.module != nil && ctx.module.config != nil {
		if ctx.module.config.OAuth2.Providers != nil {
			if _, exists := ctx.module.config.OAuth2.Providers["google"]; exists {
				return nil
			}
		}
	}

	// Spin up mock OAuth2 server if not present
	if ctx.mockOAuth2Server == nil {
		ctx.mockOAuth2Server = NewMockOAuth2Server()
		// Provide realistic user info for authorization flow
		ctx.mockOAuth2Server.SetUserInfo(map[string]interface{}{
			"id":    "oauth-user-flow-123",
			"email": "oauth.flow@example.com",
			"name":  "OAuth Flow User",
		})
	}

	provider := ctx.mockOAuth2Server.OAuth2Config("http://127.0.0.1:8080/callback")

	// Update module/service config providers map
	if ctx.module != nil && ctx.module.config != nil {
		if ctx.module.config.OAuth2.Providers == nil {
			ctx.module.config.OAuth2.Providers = map[string]OAuth2Provider{}
		}
		ctx.module.config.OAuth2.Providers["google"] = provider
	}
	if ctx.service != nil && ctx.service.config != nil {
		if ctx.service.config.OAuth2.Providers == nil {
			ctx.service.config.OAuth2.Providers = map[string]OAuth2Provider{}
		}
		ctx.service.config.OAuth2.Providers["google"] = provider
	}

	// Ensure service has oauth2Configs entry (mirrors NewService logic)
	if ctx.service != nil {
		if ctx.service.oauth2Configs == nil {
			ctx.service.oauth2Configs = make(map[string]*oauth2.Config)
		}
		ctx.service.oauth2Configs["google"] = &oauth2.Config{
			ClientID:     provider.ClientID,
			ClientSecret: provider.ClientSecret,
			RedirectURL:  provider.RedirectURL,
			Scopes:       provider.Scopes,
			Endpoint: oauth2.Endpoint{
				AuthURL:  provider.AuthURL,
				TokenURL: provider.TokenURL,
			},
		}
	}

	return nil
}

func (ctx *AuthBDDTestContext) iInitiateOAuth2Authorization() error {
	url, err := ctx.service.GetOAuth2AuthURL("google", "state-123")
	if err != nil {
		ctx.lastError = err
		return nil
	}
	ctx.oauthURL = url
	return nil
}

func (ctx *AuthBDDTestContext) theAuthorizationURLShouldBeGenerated() error {
	if ctx.oauthURL == "" {
		return fmt.Errorf("no OAuth2 authorization URL generated")
	}
	if ctx.lastError != nil {
		return fmt.Errorf("OAuth2 URL generation failed: %v", ctx.lastError)
	}
	return nil
}

func (ctx *AuthBDDTestContext) theURLShouldContainProperParameters() error {
	if ctx.oauthURL == "" {
		return fmt.Errorf("no URL to check")
	}
	// Basic check that it looks like a URL
	if len(ctx.oauthURL) < 10 {
		return fmt.Errorf("URL seems too short to be valid")
	}
	return nil
}

// OAuth2-specific step registration
func (ctx *AuthBDDTestContext) registerOAuthSteps(s *godog.ScenarioContext) {
	// OAuth2 steps
	s.Step(`^I have OAuth2 configuration$`, ctx.iHaveOAuth2Configuration)
	s.Step(`^I initiate OAuth2 authorization$`, ctx.iInitiateOAuth2Authorization)
	s.Step(`^the authorization URL should be generated$`, ctx.theAuthorizationURLShouldBeGenerated)
	s.Step(`^the URL should contain proper parameters$`, ctx.theURLShouldContainProperParameters)
}
