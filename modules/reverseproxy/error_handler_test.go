package reverseproxy

import (
	"errors"
	"net/http"
	"testing"
)

// TestClassifyProxyError verifies the error classification logic
func TestClassifyProxyError(t *testing.T) {
	m := &ReverseProxyModule{}

	tests := []struct {
		name           string
		err            error
		expectedStatus int
		expectedMsg    string
	}{
		{
			name:           "nil error",
			err:            nil,
			expectedStatus: http.StatusInternalServerError,
			expectedMsg:    "Internal server error",
		},
		{
			name:           "context deadline exceeded",
			err:            errors.New("context deadline exceeded"),
			expectedStatus: http.StatusGatewayTimeout,
			expectedMsg:    "Gateway timeout",
		},
		{
			name:           "timeout error",
			err:            errors.New("request timeout"),
			expectedStatus: http.StatusGatewayTimeout,
			expectedMsg:    "Gateway timeout",
		},
		{
			name:           "deadline error",
			err:            errors.New("deadline reached"),
			expectedStatus: http.StatusGatewayTimeout,
			expectedMsg:    "Gateway timeout",
		},
		{
			name:           "connection refused",
			err:            errors.New("connection refused"),
			expectedStatus: http.StatusBadGateway,
			expectedMsg:    "Backend service unavailable",
		},
		{
			name:           "no such host",
			err:            errors.New("no such host"),
			expectedStatus: http.StatusBadGateway,
			expectedMsg:    "Backend service unavailable",
		},
		{
			name:           "generic error",
			err:            errors.New("something went wrong"),
			expectedStatus: http.StatusInternalServerError,
			expectedMsg:    "Internal server error",
		},
		{
			name:           "case insensitive timeout",
			err:            errors.New("Request TIMEOUT occurred"),
			expectedStatus: http.StatusGatewayTimeout,
			expectedMsg:    "Gateway timeout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, msg := m.classifyProxyError(tt.err)

			if status != tt.expectedStatus {
				t.Errorf("classifyProxyError() status = %v, want %v", status, tt.expectedStatus)
			}

			if msg != tt.expectedMsg {
				t.Errorf("classifyProxyError() message = %v, want %v", msg, tt.expectedMsg)
			}
		})
	}
}
