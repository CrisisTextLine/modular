package database

import (
	"fmt"
	"net/url"
	"strings"
)

// replaceDSNPassword replaces the password in a DSN with the provided token
// This is a test helper function used to validate DSN manipulation logic
func replaceDSNPassword(dsn, token string) (string, error) {
	if strings.Contains(dsn, "://") {
		// URL-style DSN
		// Handle potential special characters in password by preprocessing
		preprocessedDSN, err := preprocessDSNForParsing(dsn)
		if err != nil {
			return "", fmt.Errorf("failed to preprocess DSN: %w", err)
		}

		u, err := url.Parse(preprocessedDSN)
		if err != nil {
			return "", fmt.Errorf("failed to parse DSN URL: %w", err)
		}

		if u.User != nil {
			username := u.User.Username()
			u.User = url.UserPassword(username, token)
		} else {
			return "", ErrNoUserInfoInDSN
		}

		return u.String(), nil
	}

	// Key-value style DSN
	parts := strings.Fields(dsn)
	var result []string
	passwordReplaced := false

	for _, part := range parts {
		if strings.HasPrefix(part, "password=") {
			result = append(result, "password="+token)
			passwordReplaced = true
		} else {
			result = append(result, part)
		}
	}

	if !passwordReplaced {
		result = append(result, "password="+token)
	}

	return strings.Join(result, " "), nil
}
