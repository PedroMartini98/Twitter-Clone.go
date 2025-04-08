package auth

import (
	"net/http"
	"testing"
)

func TestGetBearerToken(t *testing.T) {
	tests := []struct {
		name       string
		headers    http.Header
		wantToken  string
		wantErr    bool
		wantErrMsg string
	}{
		{
			name: "valid bearer token",
			headers: http.Header{
				"Authorization": []string{"Bearer abc123"},
			},
			wantToken: "abc123",
			wantErr:   false,
		},
		{
			name:       "missing header",
			headers:    http.Header{},
			wantErr:    true,
			wantErrMsg: "missing Authorization header",
		},
		{
			name: "no bearer prefix",
			headers: http.Header{
				"Authorization": []string{"abc123"},
			},
			wantErr:    true,
			wantErrMsg: "invalid Authorization header: missing Bearer prefix",
		},
		{
			name: "empty token",
			headers: http.Header{
				"Authorization": []string{"Bearer "},
			},
			wantErr:    true,
			wantErrMsg: "invalid Authorization header: empty token",
		},
		{
			name: "extra whitespace",
			headers: http.Header{
				"Authorization": []string{"Bearer   xyz789   "},
			},
			wantToken: "xyz789",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotToken, err := GetBearerToken(tt.headers)
			if tt.wantErr {
				if err == nil {
					t.Errorf("GetBearerToken() error = nil, want error %q", tt.wantErrMsg)
					return
				}
				if err.Error() != tt.wantErrMsg {
					t.Errorf("GetBearerToken() error = %q, want %q", err.Error(), tt.wantErrMsg)
				}
				return
			}
			if err != nil {
				t.Errorf("GetBearerToken() unexpected error: %v", err)
				return
			}
			if gotToken != tt.wantToken {
				t.Errorf("GetBearerToken() got = %q, want %q", gotToken, tt.wantToken)
			}
		})
	}
}
