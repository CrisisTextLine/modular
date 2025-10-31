package database

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

var (
	ErrExtractEndpointFailed = errors.New("could not extract endpoint from DSN")
	ErrNoUserInfoInDSN       = errors.New("no user information in DSN to replace password")
)

// extractEndpointFromDSN extracts the database endpoint from a DSN
func extractEndpointFromDSN(dsn string) (string, error) {
	// Handle different DSN formats
	if strings.Contains(dsn, "://") {
		// URL-style DSN (e.g., postgres://user:password@host:port/database)
		// Handle potential special characters in password by preprocessing
		preprocessedDSN, err := preprocessDSNForParsing(dsn)
		if err != nil {
			return "", fmt.Errorf("failed to preprocess DSN: %w", err)
		}

		u, err := url.Parse(preprocessedDSN)
		if err != nil {
			return "", fmt.Errorf("failed to parse DSN URL: %w", err)
		}
		return u.Host, nil
	}

	// Key-value style DSN (e.g., host=localhost port=5432 user=postgres)
	parts := strings.Fields(dsn)
	for _, part := range parts {
		if strings.HasPrefix(part, "host=") {
			host := strings.TrimPrefix(part, "host=")
			// Look for port in the same DSN
			for _, p := range parts {
				if strings.HasPrefix(p, "port=") {
					port := strings.TrimPrefix(p, "port=")
					return host + ":" + port, nil
				}
			}
			return host + ":5432", nil // Default PostgreSQL port
		}
	}

	return "", ErrExtractEndpointFailed
}

// preprocessDSNForParsing handles special characters in passwords by URL-encoding them
func preprocessDSNForParsing(dsn string) (string, error) {
	// Find the pattern: ://username:password@host
	protocolEnd := strings.Index(dsn, "://")
	if protocolEnd == -1 {
		return dsn, nil // Not a URL-style DSN
	}

	// Find the start of credentials (after ://)
	credentialsStart := protocolEnd + 3

	// Find the end of credentials (before @host)
	// We need to find the correct @ that separates credentials from host
	// Look for the pattern @host:port or @host/path or @host (end of string)
	remainingDSN := dsn[credentialsStart:]

	// Find the @ that is followed by a valid hostname pattern
	// A hostname should not contain most special characters that would be in a password
	// Search from right to left to find the last @ that's followed by a hostname
	var atIndex = -1
	for i := len(remainingDSN) - 1; i >= 0; i-- {
		if remainingDSN[i] == '@' {
			// Check if what follows looks like a hostname
			hostPart := remainingDSN[i+1:]
			if len(hostPart) > 0 && looksLikeHostname(hostPart) {
				atIndex = i
				break
			}
		}
	}

	if atIndex == -1 {
		return dsn, nil // No credentials
	}

	// Extract the credentials part
	credentialsEnd := credentialsStart + atIndex
	credentials := dsn[credentialsStart:credentialsEnd]

	// Find the colon that separates username from password
	colonIndex := strings.Index(credentials, ":")
	if colonIndex == -1 {
		return dsn, nil // No password
	}

	// Extract username and password
	username := credentials[:colonIndex]
	password := credentials[colonIndex+1:]

	// Check if password is already URL-encoded
	// A properly URL-encoded password should contain % characters followed by hex digits
	isAlreadyEncoded := strings.Contains(password, "%") && func() bool {
		// Check if it contains URL-encoded patterns like %20, %21, etc.
		for i := 0; i < len(password)-2; i++ {
			if password[i] == '%' {
				// Check if the next two characters are hex digits
				if len(password) > i+2 {
					c1, c2 := password[i+1], password[i+2]
					if isHexDigit(c1) && isHexDigit(c2) {
						return true
					}
				}
			}
		}
		return false
	}()

	if isAlreadyEncoded {
		// Password is already encoded, return as-is
		return dsn, nil
	}

	// URL-encode the password
	encodedPassword := url.QueryEscape(password)

	// Reconstruct the DSN with encoded password
	encodedDSN := dsn[:credentialsStart] + username + ":" + encodedPassword + dsn[credentialsEnd:]

	return encodedDSN, nil
}

// isHexDigit checks if a character is a hexadecimal digit
func isHexDigit(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'A' && c <= 'F') || (c >= 'a' && c <= 'f')
}

// looksLikeHostname checks if a string looks like a hostname
func looksLikeHostname(hostPart string) bool {
	// Split by / to get just the host:port part (before any path)
	parts := strings.SplitN(hostPart, "/", 2)
	hostAndPort := parts[0]

	// Split by ? to get just the host:port part (before any query params)
	parts = strings.SplitN(hostAndPort, "?", 2)
	hostAndPort = parts[0]

	if len(hostAndPort) == 0 {
		return false
	}

	// Check if it contains characters that are unlikely to be in hostnames
	// but common in passwords
	for _, char := range hostAndPort {
		// These characters are not typically found in hostnames
		if char == '!' || char == '#' || char == '$' || char == '%' ||
			char == '^' || char == '&' || char == '*' || char == '(' ||
			char == ')' || char == '+' || char == '=' || char == '[' ||
			char == ']' || char == '{' || char == '}' || char == '|' ||
			char == ';' || char == '\'' || char == '"' || char == ',' ||
			char == '<' || char == '>' || char == '\\' {
			return false
		}
	}

	// Additional checks: hostname should contain at least one dot or be localhost
	// and should not start with special characters
	return (strings.Contains(hostAndPort, ".") || hostAndPort == "localhost" ||
		strings.Contains(hostAndPort, ":")) &&
		(len(hostAndPort) > 0 && (hostAndPort[0] >= 'a' && hostAndPort[0] <= 'z') ||
			(hostAndPort[0] >= 'A' && hostAndPort[0] <= 'Z') ||
			(hostAndPort[0] >= '0' && hostAndPort[0] <= '9'))
}

// replaceDSNPassword replaces the password in a DSN with the provided token
//
//nolint:unused // Used in test files
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
